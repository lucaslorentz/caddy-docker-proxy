package plugin

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/assert"
)

var caddyContainerID = "container-id"
var caddyNetworkID = "network-id"

const skipCaddyfileText = "[INFO] Skipping default CaddyFile because no path is set\n"

func init() {
	log.SetOutput(ioutil.Discard)
}

func fmtLabel(s string) string {
	return fmt.Sprintf(s, defaultLabelPrefix)
}

func TestUseTemplatesToGenerateEmptyValues(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s"):                   "{{.Spec.Name}}.testdomain.com",
						fmtLabel("%s.proxy"):             "/ {{.Spec.Name}}:5000/api",
						fmtLabel("%s.proxy.transparent"): "{{nil}}",
						fmtLabel("%s.proxy.websocket"):   "{{nil}}",
						fmtLabel("%s.gzip"):              "{{nil}}",
					},
				},
			},
			Endpoint: swarm.Endpoint{
				VirtualIPs: []swarm.EndpointVirtualIP{
					swarm.EndpointVirtualIP{
						NetworkID: caddyNetworkID,
					},
				},
			},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"  gzip\n" +
		"  proxy / service:5000/api {\n" +
		"    transparent\n" +
		"    websocket\n" +
		"  }\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestAddDockerConfigContent(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ConfigsData = []swarm.Config{
		swarm.Config{
			ID: "CONFIG-ID",
			Spec: swarm.ConfigSpec{
				Annotations: swarm.Annotations{
					Labels: map[string]string{
						fmtLabel("%s"): "",
					},
				},
				Data: []byte(
					"example.com {\n" +
						"  tls off+\n" +
						"}",
				),
			},
		},
	}

	const expectedCaddyfile = "example.com {\n" +
		"  tls off+\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestIgnoreLabelsWithoutCaddyPrefix(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						"caddy_version":  "0.11.0",
						"caddyversion":   "0.11.0",
						"caddy_.version": "0.11.0",
						"version_caddy":  "0.11.0",
					},
				},
			},
			Endpoint: swarm.Endpoint{
				VirtualIPs: []swarm.EndpointVirtualIP{
					swarm.EndpointVirtualIP{
						NetworkID: caddyNetworkID,
					},
				},
			},
		},
	}

	const expectedCaddyfile = ""

	testGeneration(t, dockerClient, true, true, expectedCaddyfile, skipCaddyfileText)
}

func testGeneration(
	t *testing.T,
	dockerClient DockerClient,
	proxyServiceTasks bool,
	validateNetwork bool,
	expectedCaddyfile string,
	expectedLogs string,
) {
	dockerUtils := createDockerUtilsMock()

	generator := CreateGenerator(dockerClient, dockerUtils, &GeneratorOptions{
		labelPrefix:       defaultLabelPrefix,
		proxyServiceTasks: proxyServiceTasks,
		validateNetwork:   validateNetwork,
	})

	caddyfileBytes, logs := generator.GenerateCaddyFile()
	assert.Equal(t, expectedCaddyfile, string(caddyfileBytes))
	assert.Equal(t, expectedLogs, logs)
}

func createBasicDockerClientMock() *dockerClientMock {
	return &dockerClientMock{
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
			caddyContainerID: types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{
						"overlay": &network.EndpointSettings{
							NetworkID: caddyNetworkID,
						},
					},
				},
			},
		},
		NetworkInspectData: map[string]types.NetworkResource{
			caddyNetworkID: types.NetworkResource{
				Ingress: false,
			},
		},
	}
}

func createDockerUtilsMock() *dockerUtilsMock {
	return &dockerUtilsMock{
		MockGetCurrentContainerID: func() (string, error) {
			return caddyContainerID, nil
		},
	}
}

type dockerClientMock struct {
	ContainersData       []types.Container
	ServicesData         []swarm.Service
	ConfigsData          []swarm.Config
	TasksData            []swarm.Task
	InfoData             types.Info
	ContainerInspectData map[string]types.ContainerJSON
	NetworkInspectData   map[string]types.NetworkResource
}

func (mock *dockerClientMock) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	return mock.ContainersData, nil
}

func (mock *dockerClientMock) ServiceList(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {
	return mock.ServicesData, nil
}

func (mock *dockerClientMock) TaskList(ctx context.Context, options types.TaskListOptions) ([]swarm.Task, error) {
	matchingTasks := []swarm.Task{}
	for _, task := range mock.TasksData {
		if !options.Filters.Match("service", task.ServiceID) {
			continue
		}
		if !options.Filters.Match("desired-state", string(task.DesiredState)) {
			continue
		}
		matchingTasks = append(matchingTasks, task)
	}
	return matchingTasks, nil
}

func (mock *dockerClientMock) Info(ctx context.Context) (types.Info, error) {
	return mock.InfoData, nil
}

func (mock *dockerClientMock) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return mock.ContainerInspectData[containerID], nil
}

func (mock *dockerClientMock) NetworkInspect(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, error) {
	return mock.NetworkInspectData[networkID], nil
}

func (mock *dockerClientMock) ConfigList(ctx context.Context, options types.ConfigListOptions) ([]swarm.Config, error) {
	return mock.ConfigsData, nil
}

func (mock *dockerClientMock) ConfigInspectWithRaw(ctx context.Context, id string) (swarm.Config, []byte, error) {
	for _, config := range mock.ConfigsData {
		if config.ID == id {
			return config, nil, nil
		}
	}
	return swarm.Config{}, nil, nil
}

type dockerUtilsMock struct {
	MockGetCurrentContainerID func() (string, error)
}

// GetCurrentContainerID returns the id of the container running this application
func (mock *dockerUtilsMock) GetCurrentContainerID() (string, error) {
	return mock.MockGetCurrentContainerID()
}
