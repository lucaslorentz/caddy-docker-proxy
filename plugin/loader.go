package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/lucaslorentz/caddy-docker-proxy/plugin/v2/config"
	"github.com/lucaslorentz/caddy-docker-proxy/plugin/v2/docker"
	"github.com/lucaslorentz/caddy-docker-proxy/plugin/v2/generator"
)

// DockerLoader generates caddy files from docker swarm information
type DockerLoader struct {
	options        *config.Options
	initialized    bool
	dockerClient   docker.Client
	generator      *generator.CaddyfileGenerator
	timer          *time.Timer
	skipEvents     bool
	lastCaddyfile  []byte
	lastLogs       string
	lastAppsConfig []byte
	updatedServers map[string]struct{}
}

// CreateDockerLoader creates a docker loader
func CreateDockerLoader(options *config.Options) *DockerLoader {
	return &DockerLoader{
		options:        options,
		updatedServers: map[string]struct{}{},
	}
}

// Start docker loader
func (dockerLoader *DockerLoader) Start() error {
	if !dockerLoader.initialized {
		dockerLoader.initialized = true

		dockerClient, err := client.NewEnvClient()
		if err != nil {
			log.Printf("[ERROR] Docker connection failed: %s", err)
			return err
		}

		dockerPing, err := dockerClient.Ping(context.Background())
		if err != nil {
			log.Printf("[ERROR] Docker ping failed: %s", err)
			return err
		}

		dockerClient.NegotiateAPIVersionPing(dockerPing)

		wrappedClient := docker.WrapClient(dockerClient)

		dockerLoader.dockerClient = wrappedClient
		dockerLoader.generator = generator.CreateGenerator(
			wrappedClient,
			docker.CreateUtils(),
			dockerLoader.options,
		)

		log.Printf("[INFO] CaddyfilePath: %v", dockerLoader.options.CaddyfilePath)
		log.Printf("[INFO] LabelPrefix: %v", dockerLoader.options.LabelPrefix)
		log.Printf("[INFO] PollingInterval: %v", dockerLoader.options.PollingInterval)
		log.Printf("[INFO] ProcessCaddyfile: %v", dockerLoader.options.ProcessCaddyfile)
		log.Printf("[INFO] ProxyServiceTasks: %v", dockerLoader.options.ProxyServiceTasks)
		log.Printf("[INFO] ValidateNetwork: %v", dockerLoader.options.ValidateNetwork)

		dockerLoader.timer = time.AfterFunc(0, func() {
			dockerLoader.update()
		})

		go dockerLoader.monitorEvents()
	}

	return nil
}

func (dockerLoader *DockerLoader) monitorEvents() {
	args := filters.NewArgs()
	args.Add("scope", "swarm")
	args.Add("scope", "local")
	args.Add("type", "service")
	args.Add("type", "container")
	args.Add("type", "config")

	eventsChan, errorChan := dockerLoader.dockerClient.Events(context.Background(), types.EventsOptions{
		Filters: args,
	})

	for {
		select {
		case event := <-eventsChan:
			if dockerLoader.skipEvents {
				continue
			}

			update := (event.Type == "container" && event.Action == "create") ||
				(event.Type == "container" && event.Action == "start") ||
				(event.Type == "container" && event.Action == "stop") ||
				(event.Type == "container" && event.Action == "die") ||
				(event.Type == "container" && event.Action == "destroy") ||
				(event.Type == "service" && event.Action == "create") ||
				(event.Type == "service" && event.Action == "update") ||
				(event.Type == "service" && event.Action == "remove") ||
				(event.Type == "config" && event.Action == "create") ||
				(event.Type == "config" && event.Action == "remove")

			if update {
				dockerLoader.skipEvents = true
				dockerLoader.timer.Reset(100 * time.Millisecond)
			}
		case err := <-errorChan:
			log.Println(err)
		}
	}
}

func (dockerLoader *DockerLoader) update() bool {
	dockerLoader.timer.Reset(dockerLoader.options.PollingInterval)
	dockerLoader.skipEvents = false

	caddyfile, logs, controlledServers := dockerLoader.generator.GenerateCaddyfile()

	caddyfileChanged := !bytes.Equal(dockerLoader.lastCaddyfile, caddyfile)
	logsChanged := dockerLoader.lastLogs != logs

	dockerLoader.lastCaddyfile = caddyfile
	dockerLoader.lastLogs = logs

	if logsChanged || caddyfileChanged {
		log.Print(logs)
	}

	if caddyfileChanged {
		log.Printf("[INFO] New Caddyfile:\n%s", caddyfile)

		adapter := caddyconfig.GetAdapter("caddyfile")

		configJSON, warn, err := adapter.Adapt(caddyfile, nil)

		if warn != nil {
			log.Printf("[WARNING] Caddyfile to json warning: %v", warn)
		}

		if err != nil {
			log.Printf("[ERROR] Failed to convert caddyfile into json config: %s", err)
		}

		log.Printf("[INFO] New Config JSON:\n%s", configJSON)

		var full map[string]interface{}
		json.Unmarshal(configJSON, &full)
		apps, _ := full["apps"]
		appsConfig, _ := json.Marshal(apps)

		dockerLoader.lastAppsConfig = appsConfig
		dockerLoader.updatedServers = map[string]struct{}{}
	}

	var wg sync.WaitGroup
	for _, server := range controlledServers {
		wg.Add(1)
		go dockerLoader.updateServer(&wg, server)
	}
	wg.Wait()

	return true
}

func (dockerLoader *DockerLoader) updateServer(wg *sync.WaitGroup, server string) {
	defer wg.Done()

	if _, isUpToDate := dockerLoader.updatedServers[server]; isUpToDate {
		return
	}

	log.Printf("[INFO] Sending configuration to %v", server)

	resp, err := http.Post("http://"+server+":2019/config/apps", "application/json", bytes.NewBuffer(dockerLoader.lastAppsConfig))
	if err != nil {
		log.Printf("[ERROR] Failed to send configuration to %v: %s", server, err)
		return
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read response from %v: %s", server, err)
		return
	}

	if resp.StatusCode != 200 {
		log.Printf("[ERROR] Error response from %v: %v - %s", server, resp.StatusCode, bodyBytes)
		return
	}

	dockerLoader.updatedServers[server] = struct{}{}

	log.Printf("[INFO] Successfully configured %v", server)
}
