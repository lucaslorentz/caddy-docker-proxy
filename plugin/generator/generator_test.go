package generator

import (
	"fmt"
	"io/ioutil"
	"log"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/lucaslorentz/caddy-docker-proxy/plugin/v2/config"
	"github.com/lucaslorentz/caddy-docker-proxy/plugin/v2/docker"
	"github.com/stretchr/testify/assert"
)

var caddyContainerID = "container-id"
var caddyNetworkID = "network-id"

const skipCaddyfileText = "[INFO] Skipping default Caddyfile because no path is set\n"

func init() {
	log.SetOutput(ioutil.Discard)
}

func fmtLabel(s string) string {
	return fmt.Sprintf(s, DefaultLabelPrefix)
}

func TestAddDockerConfigContent(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ConfigsData = []swarm.Config{
		{
			ID: "CONFIG-ID",
			Spec: swarm.ConfigSpec{
				Annotations: swarm.Annotations{
					Labels: map[string]string{
						fmtLabel("%s"): "",
					},
				},
				Data: []byte(
					"example.com {\n" +
						"	tls off+\n" +
						"}",
				),
			},
		},
	}

	const expectedCaddyfile = "example.com {\n" +
		"	tls off+\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestIgnoreLabelsWithoutCaddyPrefix(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						"caddy_version":  "2.0.0",
						"caddyversion":   "2.0.0",
						"caddy_.version": "2.0.0",
						"version_caddy":  "2.0.0",
					},
				},
			},
			Endpoint: swarm.Endpoint{
				VirtualIPs: []swarm.EndpointVirtualIP{
					{
						NetworkID: caddyNetworkID,
					},
				},
			},
		},
	}

	const expectedCaddyfile = "# Empty caddyfile"

	testGeneration(t, dockerClient, true, true, expectedCaddyfile, skipCaddyfileText)
}

func testGeneration(
	t *testing.T,
	dockerClient docker.Client,
	proxyServiceTasks bool,
	validateNetwork bool,
	expectedCaddyfile string,
	expectedLogs string,
) {
	dockerUtils := createDockerUtilsMock()

	generator := CreateGenerator(dockerClient, dockerUtils, &config.Options{
		LabelPrefix:       DefaultLabelPrefix,
		ProxyServiceTasks: proxyServiceTasks,
		ValidateNetwork:   validateNetwork,
	})

	caddyfileBytes, logs, _ := generator.GenerateCaddyfile()
	assert.Equal(t, expectedCaddyfile, string(caddyfileBytes))
	assert.Equal(t, expectedLogs, logs)
}

func createBasicDockerClientMock() *docker.ClientMock {
	return &docker.ClientMock{
		ContainersData: []types.Container{},
		ServicesData:   []swarm.Service{},
		ConfigsData:    []swarm.Config{},
		TasksData:      []swarm.Task{},
		InfoData: types.Info{
			Swarm: swarm.Info{
				LocalNodeState: swarm.LocalNodeStateActive,
			},
		},
		ContainerInspectData: map[string]types.ContainerJSON{
			caddyContainerID: {
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{
						"overlay": {
							NetworkID: caddyNetworkID,
						},
					},
				},
			},
		},
		NetworkInspectData: map[string]types.NetworkResource{
			caddyNetworkID: {
				Ingress: false,
			},
		},
	}
}

func createDockerUtilsMock() *docker.UtilsMock {
	return &docker.UtilsMock{
		MockGetCurrentContainerID: func() (string, error) {
			return caddyContainerID, nil
		},
	}
}
