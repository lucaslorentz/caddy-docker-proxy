package plugin

import (
	"bytes"
	"log"
	"time"

	"github.com/mholt/caddy"
)

// DockerLoader generates caddy files from docker swarm information
type DockerLoader struct {
	Input caddy.CaddyfileInput
}

// CreateDockerLoader creates a docker loader
func CreateDockerLoader() *DockerLoader {
	loader := DockerLoader{
		Input: caddy.CaddyfileInput{
			ServerTypeName: "http",
		},
	}
	loader.updateInput()
	loader.scheduleUpdate()
	return &loader
}

func (dockerLoader *DockerLoader) scheduleUpdate() {
	time.AfterFunc(10*time.Second, func() {
		if dockerLoader.updateInput() {
			ReloadCaddy()
		}
		dockerLoader.scheduleUpdate()
	})
}

func (dockerLoader *DockerLoader) updateInput() bool {
	newContents := GenerateCaddyFile()

	if bytes.Equal(dockerLoader.Input.Contents, newContents) {
		return false
	}

	dockerLoader.Input.Contents = newContents

	log.Printf("[INFO] New CaddyFile:\n%s", dockerLoader.Input.Contents)

	return true
}

// Load returns the current caddy file input
func (dockerLoader *DockerLoader) Load(serverType string) (caddy.Input, error) {
	if serverType != "http" {
		return nil, nil
	}
	return dockerLoader.Input, nil
}
