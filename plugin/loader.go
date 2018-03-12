package plugin

import (
	"bytes"
	"context"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/mholt/caddy"
)

const poolInterval = 10 * time.Second

// DockerLoader generates caddy files from docker swarm information
type DockerLoader struct {
	initialized  bool
	dockerClient *client.Client
	timer        *time.Timer
	skipEvents   bool
	Input        caddy.CaddyfileInput
}

// CreateDockerLoader creates a docker loader
func CreateDockerLoader() *DockerLoader {
	return &DockerLoader{
		Input: caddy.CaddyfileInput{
			ServerTypeName: "http",
		},
	}
}

// Load returns the current caddy file input
func (dockerLoader *DockerLoader) Load(serverType string) (caddy.Input, error) {
	if serverType != "http" {
		return nil, nil
	}
	if !dockerLoader.initialized {
		dockerLoader.initialized = true

		dockerClient, err := client.NewEnvClient()
		if err != nil {
			log.Printf("Docker connection failed: %v", err)
			return nil, nil
		}

		dockerPing, err := dockerClient.Ping(context.Background())
		if err != nil {
			log.Printf("Docker ping failed: %v", err)
			return nil, nil
		}

		dockerClient.NegotiateAPIVersionPing(dockerPing)

		dockerLoader.dockerClient = dockerClient

		dockerLoader.timer = time.AfterFunc(poolInterval, func() {
			dockerLoader.update(true)
		})

		dockerLoader.update(false)

		go dockerLoader.monitorEvents()
	}
	return dockerLoader.Input, nil
}

func (dockerLoader *DockerLoader) monitorEvents() {
	args := filters.NewArgs()
	args.Add("scope", "swarm")
	args.Add("scope", "local")
	args.Add("type", "service")
	args.Add("type", "container")

	eventsChan, errorChan := dockerLoader.dockerClient.Events(context.Background(), types.EventsOptions{
		Filters: args,
	})

	for {
		select {
		case event := <-eventsChan:
			if dockerLoader.skipEvents {
				continue
			}

			update := (event.Type == "container" && event.Action == "start") ||
				(event.Type == "container" && event.Action == "stop") ||
				(event.Type == "service" && event.Action == "create") ||
				(event.Type == "service" && event.Action == "update") ||
				(event.Type == "service" && event.Action == "remove")

			if update {
				dockerLoader.skipEvents = true
				dockerLoader.timer.Reset(100 * time.Millisecond)
			}
		case err := <-errorChan:
			log.Println(err)
		}
	}
}

func (dockerLoader *DockerLoader) update(reloadIfChanged bool) bool {
	dockerLoader.timer.Reset(poolInterval)
	dockerLoader.skipEvents = false

	newContents := GenerateCaddyFile(dockerLoader.dockerClient)

	if bytes.Equal(dockerLoader.Input.Contents, newContents) {
		return false
	}

	dockerLoader.Input.Contents = newContents

	log.Printf("[INFO] New CaddyFile:\n%s", dockerLoader.Input.Contents)

	if reloadIfChanged {
		ReloadCaddy()
	}

	return true
}
