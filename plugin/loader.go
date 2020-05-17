package plugin

import (
	"bytes"
	"context"
	"log"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/plugin/config"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/plugin/docker"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/plugin/generator"
)

// DockerLoader generates caddy files from docker swarm information
type DockerLoader struct {
	options       *config.Options
	initialized   bool
	dockerClient  docker.Client
	generator     *generator.CaddyfileGenerator
	timer         *time.Timer
	skipEvents    bool
	lastCaddyfile []byte
	lastLogs      string
}

// CreateDockerLoader creates a docker loader
func CreateDockerLoader(options *config.Options) *DockerLoader {
	return &DockerLoader{
		options: options,
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

		log.Printf("[INFO] CaddyFilePath: %v", dockerLoader.options.CaddyFilePath)
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

	caddyfile, logs := dockerLoader.generator.GenerateCaddyFile()

	caddyfileChanged := !bytes.Equal(dockerLoader.lastCaddyfile, caddyfile)
	logsChanged := dockerLoader.lastLogs != logs

	dockerLoader.lastCaddyfile = caddyfile
	dockerLoader.lastLogs = logs

	if logsChanged || caddyfileChanged {
		log.Print(logs)
	}

	if !caddyfileChanged {
		return false
	}

	if len(caddyfile) == 0 {
		caddyfile = []byte("# Empty caddyfile")
	}

	log.Printf("[INFO] New CaddyFile:\n%s", caddyfile)

	adapter := caddyconfig.GetAdapter("caddyfile")

	configJSON, warn, err := adapter.Adapt(caddyfile, nil)

	if warn != nil {
		log.Printf("[WARN] Caddyfile to json warning: %v", warn)
	}

	if err != nil {
		log.Printf("[ERROR] Failed to convert caddyfile to json config: %s", err)
	}

	log.Printf("[INFO] New Config JSON:\n%s", configJSON)

	err = caddy.Load(configJSON, false)

	if err != nil {
		log.Printf("[ERROR] Failed to load caddyfile: %s", err)
		return false
	}

	return true
}
