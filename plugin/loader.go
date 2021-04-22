package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/caddyserver/caddy/v2"
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
	options         *config.Options
	initialized     bool
	dockerClient    docker.Client
	generator       *generator.CaddyfileGenerator
	timer           *time.Timer
	skipEvents      bool
	lastCaddyfile   []byte
	lastLogs        string
	lastJSONConfig  []byte
	lastVersion     int64
	serversVersions *StringInt64CMap
	serversUpdating *StringBoolCMap
}

// CreateDockerLoader creates a docker loader
func CreateDockerLoader(options *config.Options) *DockerLoader {
	return &DockerLoader{
		options:         options,
		serversVersions: newStringInt64CMap(),
		serversUpdating: newStringBoolCMap(),
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
		log.Printf("[INFO] IngressNetworks: %v", dockerLoader.options.IngressNetworks)

		dockerLoader.timer = time.AfterFunc(0, func() {
			dockerLoader.update()
		})

		go dockerLoader.monitorEvents()
	}

	return nil
}

func (dockerLoader *DockerLoader) monitorEvents() {
	for {
		dockerLoader.listenEvents()
		time.Sleep(30 * time.Second)
	}
}

func (dockerLoader *DockerLoader) listenEvents() {
	args := filters.NewArgs()
	args.Add("scope", "swarm")
	args.Add("scope", "local")
	args.Add("type", "service")
	args.Add("type", "container")
	args.Add("type", "config")

	context, cancel := context.WithCancel(context.Background())

	eventsChan, errorChan := dockerLoader.dockerClient.Events(context, types.EventsOptions{
		Filters: args,
	})

	log.Printf("[INFO] Connecting to docker events")

ListenEvents:
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
			cancel()
			if err != nil {
				log.Printf("[ERROR] Docker events error: %v", err)
			}
			break ListenEvents
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
			return false
		}

		log.Printf("[INFO] New Config JSON:\n%s", configJSON)

		dockerLoader.lastJSONConfig = configJSON
		dockerLoader.lastVersion++
	}

	var errorCounter uint64
	var wg sync.WaitGroup
	for _, server := range controlledServers {
		wg.Add(1)
		go func() {
			if (!(dockerLoader.updateServer(server))) {
   				atomic.AddUint64(&errorCounter, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if errorCounter > 0 {
		server := "localhost"
		log.Printf("[INFO] Retrying after failure on %v", server)
		updateServer(server)
	}
	return true
}

func (dockerLoader *DockerLoader) updateServer(server string) bool {

	// Skip servers that are being updated already
	if dockerLoader.serversUpdating.Get(server) {
		return true
	}

	// Flag and unflag updating
	dockerLoader.serversUpdating.Set(server, true)
	defer dockerLoader.serversUpdating.Delete(server)

	version := dockerLoader.lastVersion

	// Skip servers that already have this version
	if dockerLoader.serversVersions.Get(server) >= version {
		return true
	}

	log.Printf("[INFO] Sending configuration to %v", server)

	url := "http://" + server + ":2019/load"

	postBody, err := addAdminListen(dockerLoader.lastJSONConfig, "tcp/"+server+":2019")
	if err != nil {
		log.Printf("[ERROR] Failed to add admin listen to %v: %s", server, err)
		return false
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(postBody))
	if err != nil {
		log.Printf("[ERROR] Failed to create request to %v: %s", server, err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Printf("[ERROR] Failed to send configuration to %v: %s", server, err)
		return false
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read response from %v: %s", server, err)
		return false
	}

	if resp.StatusCode != 200 {
		log.Printf("[ERROR] Error response from %v: %v - %s", server, resp.StatusCode, bodyBytes)
		return false
	}

	dockerLoader.serversVersions.Set(server, version)

	log.Printf("[INFO] Successfully configured %v", server)
	return true
}

func addAdminListen(configJSON []byte, listen string) ([]byte, error) {
	config := &caddy.Config{}
	err := json.Unmarshal(configJSON, config)
	if err != nil {
		return nil, err
	}
	config.Admin = &caddy.AdminConfig{
		Listen: listen,
	}
	return json.Marshal(config)
}
