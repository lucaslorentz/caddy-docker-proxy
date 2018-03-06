package plugin

import (
	"bytes"
	"log"
	"time"

	"github.com/mholt/caddy"
)

// DockerLoader generates caddy files from docker swarm information
type DockerLoader struct {
	Input       caddy.CaddyfileInput
	Initialized bool
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
	if !dockerLoader.Initialized {
		dockerLoader.Initialized = true
		dockerLoader.updateInput()
		dockerLoader.scheduleUpdate()
	}
	return dockerLoader.Input, nil
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
