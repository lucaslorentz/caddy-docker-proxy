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

const skipCaddyfileText = "# Skipping default CaddyFile because no path are set\n"

func init() {
	log.SetOutput(ioutil.Discard)
}

func fmtLabel(s string) string {
	return fmt.Sprintf(s, defaultLabelPrefix)
}

func TestAddContainerWithTemplates(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		types.Container{
			Names: []string{
				"container-name",
			},
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": &network.EndpointSettings{
						IPAddress: "172.17.0.2",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):       "{{index .Names 0}}.testdomain.com",
				fmtLabel("%s.proxy"): "/ {{(index .NetworkSettings.Networks \"caddy-network\").IPAddress}}:5000/api",
			},
		},
	}

	const expected string = skipCaddyfileText +
		"container-name.testdomain.com {\n" +
		"  proxy / 172.17.0.2:5000/api\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddContainerPicksRightNetwork(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		types.Container{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"other-network": &network.EndpointSettings{
						IPAddress: "10.0.0.1",
						NetworkID: "other-network-id",
					},
					"caddy-network": &network.EndpointSettings{
						IPAddress: "172.17.0.2",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s.address"): "service.testdomain.com",
			},
		},
	}

	const expected string = skipCaddyfileText +
		"service.testdomain.com {\n" +
		"  proxy / 172.17.0.2\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddContainerWithMinimumBasicLabels(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		types.Container{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": &network.EndpointSettings{
						IPAddress: "172.17.0.2",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s.address"): "service.testdomain.com",
			},
		},
	}

	const expected string = skipCaddyfileText +
		"service.testdomain.com {\n" +
		"  proxy / 172.17.0.2\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddContainerWithAllBasicLabels(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		types.Container{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": &network.EndpointSettings{
						IPAddress: "172.17.0.2",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s.address"):        "service.testdomain.com",
				fmtLabel("%s.targetport"):     "5000",
				fmtLabel("%s.targetpath"):     "/api",
				fmtLabel("%s.targetprotocol"): "https",
			},
		},
	}

	const expected string = skipCaddyfileText +
		"service.testdomain.com {\n" +
		"  proxy / https://172.17.0.2:5000/api\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddContainerFromDifferentNetwork(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		types.Container{
			ID: "CONTAINER-ID",
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"other-network": &network.EndpointSettings{
						IPAddress: "10.0.0.1",
						NetworkID: "other-network-id",
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s.address"): "service.testdomain.com",
			},
		},
	}

	const expected string = skipCaddyfileText +
		"# Container CONTAINER-ID and caddy are not in same network\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddContainerWithMultipleConfigs(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		types.Container{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": &network.EndpointSettings{
						IPAddress: "172.17.0.2",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s_0.address"):    "service1.testdomain.com",
				fmtLabel("%s_0.targetport"): "5000",
				fmtLabel("%s_0.targetpath"): "/api",
				fmtLabel("%s_0.tls.dns"):    "route53",
				fmtLabel("%s_1.address"):    "service2.testdomain.com",
				fmtLabel("%s_1.targetport"): "5001",
				fmtLabel("%s_1.tls.dns"):    "route53",
			},
		},
	}

	const expected string = skipCaddyfileText +
		"service1.testdomain.com {\n" +
		"  proxy / 172.17.0.2:5000/api\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n" +
		"service2.testdomain.com {\n" +
		"  proxy / 172.17.0.2:5001\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddContainerWithReplicas(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		types.Container{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": &network.EndpointSettings{
						IPAddress: "172.17.0.2",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s.address"): "service.testdomain.com",
			},
		},
		types.Container{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": &network.EndpointSettings{
						IPAddress: "172.17.0.3",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s.address"): "service.testdomain.com",
			},
		},
	}

	const expected string = skipCaddyfileText +
		"service.testdomain.com {\n" +
		"  proxy / 172.17.0.2 172.17.0.3\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestDoNotMergeProxiesWithDifferentLabelKey(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		types.Container{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": &network.EndpointSettings{
						IPAddress: "172.17.0.2",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):         "service.testdomain.com",
				fmtLabel("%s.proxy_0"): "/a service-a",
			},
		},
		types.Container{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": &network.EndpointSettings{
						IPAddress: "172.17.0.3",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):         "service.testdomain.com",
				fmtLabel("%s.proxy_1"): "/b service-b",
			},
		},
	}

	const expected string = skipCaddyfileText +
		"service.testdomain.com {\n" +
		"  proxy /a service-a\n" +
		"  proxy /b service-b\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddContainersWithSnippets(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		types.Container{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": &network.EndpointSettings{
						IPAddress: "172.17.0.3",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s.address"): "service.testdomain.com",
				fmtLabel("%s.import"):  "mysnippet-1",
			},
		},
		types.Container{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": &network.EndpointSettings{
						IPAddress: "172.17.0.2",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s_1"):     "(mysnippet-1)",
				fmtLabel("%s_1.tls"): "off",
				fmtLabel("%s_2"):     "(mysnippet-2)",
				fmtLabel("%s_2.tls"): "off",
			},
		},
	}

	const expected string = skipCaddyfileText +
		"(mysnippet-1) {\n" +
		"  tls off\n" +
		"}\n" +
		"(mysnippet-2) {\n" +
		"  tls off\n" +
		"}\n" +
		"service.testdomain.com {\n" +
		"  import mysnippet-1\n" +
		"  proxy / 172.17.0.3\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddServiceWithTemplates(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s"):                    "{{.Spec.Name}}.testdomain.com",
						fmtLabel("%s.proxy"):              "/ {{.Spec.Name}}:5000/api",
						fmtLabel("%s.proxy.transparent"):  "",
						fmtLabel("%s.proxy.health_check"): "/health",
						fmtLabel("%s.proxy.websocket"):    "",
						fmtLabel("%s.gzip"):               "",
						fmtLabel("%s.basicauth"):          "/ user password",
						fmtLabel("%s.tls.dns"):            "route53",
						fmtLabel("%s.rewrite_0"):          "/path1 /path2",
						fmtLabel("%s.rewrite_1"):          "/path3 /path4",
						fmtLabel("%s.limits.header"):      "100kb",
						fmtLabel("%s.limits.body_0"):      "/path1 2mb",
						fmtLabel("%s.limits.body_1"):      "/path2 4mb",
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

	const expected string = skipCaddyfileText +
		"service.testdomain.com {\n" +
		"  basicauth / user password\n" +
		"  gzip\n" +
		"  limits {\n" +
		"    body /path1 2mb\n" +
		"    body /path2 4mb\n" +
		"    header 100kb\n" +
		"  }\n" +
		"  proxy / service:5000/api {\n" +
		"    health_check /health\n" +
		"    transparent\n" +
		"    websocket\n" +
		"  }\n" +
		"  rewrite /path1 /path2\n" +
		"  rewrite /path3 /path4\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddServiceWithMinimumBasicLabels(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s.address"): "service.testdomain.com",
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

	const expected string = skipCaddyfileText +
		"service.testdomain.com {\n" +
		"  proxy / service\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddServiceWithAllBasicLabels(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s.address"):        "service.testdomain.com",
						fmtLabel("%s.targetport"):     "5000",
						fmtLabel("%s.targetpath"):     "/api",
						fmtLabel("%s.targetprotocol"): "https",
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

	const expected string = skipCaddyfileText +
		"service.testdomain.com {\n" +
		"  proxy / https://service:5000/api\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddServiceWithMultipleConfigs(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s_0.address"):            "service1.testdomain.com",
						fmtLabel("%s_0.targetport"):         "5000",
						fmtLabel("%s_0.targetpath"):         "/api",
						fmtLabel("%s_0.proxy.health_check"): "/health",
						fmtLabel("%s_0.proxy.transparent"):  "",
						fmtLabel("%s_0.proxy.websocket"):    "",
						fmtLabel("%s_0.basicauth"):          "/ user password",
						fmtLabel("%s_0.tls.dns"):            "route53",
						fmtLabel("%s_1.address"):            "service2.testdomain.com",
						fmtLabel("%s_1.targetport"):         "5001",
						fmtLabel("%s_1.tls.dns"):            "route53",
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

	const expected string = skipCaddyfileText +
		"service1.testdomain.com {\n" +
		"  basicauth / user password\n" +
		"  proxy / service:5000/api {\n" +
		"    health_check /health\n" +
		"    transparent\n" +
		"    websocket\n" +
		"  }\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n" +
		"service2.testdomain.com {\n" +
		"  proxy / service:5001\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddServiceProxyServiceTasks(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s.address"):    "service.testdomain.com",
						fmtLabel("%s.targetport"): "5000",
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

	const expected string = skipCaddyfileText +
		"service.testdomain.com {\n" +
		"  proxy / tasks.service:5000\n" +
		"}\n"

	testGeneration(t, dockerClient, true, expected)
}

func TestAddServiceMultipleAddresses(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s.address"): "a.testdomain.com b.testdomain.com",
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

	const expected string = skipCaddyfileText +
		"a.testdomain.com b.testdomain.com {\n" +
		"  proxy / service\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAutomaticProxyDoesntOverrideCustomWithSameKey(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s.address"): "testdomain.com",
						fmtLabel("%s.proxy"):   "/ something",
						fmtLabel("%s.proxy_1"): "/api external-api",
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

	const expected string = skipCaddyfileText +
		"testdomain.com {\n" +
		"  proxy / something\n" +
		"  proxy /api external-api\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddServiceFromDifferentNetwork(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			ID: "SERVICE-ID",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s.address"): "service.testdomain.com",
					},
				},
			},
			Endpoint: swarm.Endpoint{
				VirtualIPs: []swarm.EndpointVirtualIP{
					swarm.EndpointVirtualIP{
						NetworkID: "other-network-id",
					},
				},
			},
		},
	}

	const expected string = skipCaddyfileText +
		"# Service SERVICE-ID and caddy are not in same network\n"

	testGeneration(t, dockerClient, false, expected)
}

func TestAddServiceSwarmDisable(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			ID: "SERVICE-ID",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s.address"): "service.testdomain.com",
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
	dockerClient.InfoData = types.Info{
		Swarm: swarm.Info{
			LocalNodeState: swarm.LocalNodeStateInactive,
		},
	}

	const expected string = skipCaddyfileText +
		"# Skipping services because swarm is not available\n" +
		"# Skipping configs because swarm is not available\n"

	testGeneration(t, dockerClient, false, expected)
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

	const expected string = skipCaddyfileText +
		"example.com {\n" +
		"  tls off+\n" +
		"}\n"

	testGeneration(t, dockerClient, false, expected)
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

	const expected string = skipCaddyfileText

	testGeneration(t, dockerClient, true, expected)
}

func testGeneration(
	t *testing.T,
	dockerClient DockerClient,
	proxyServiceTasks bool,
	expected string,
) {
	dockerUtils := createDockerUtilsMock()

	generator := CreateGenerator(dockerClient, dockerUtils, &GeneratorOptions{
		labelPrefix:       defaultLabelPrefix,
		proxyServiceTasks: proxyServiceTasks,
	})

	bytes := generator.GenerateCaddyFile()
	var content = string(bytes[:])
	assert.Equal(t, expected, content)
}

func createBasicDockerClientMock() *dockerClientMock {
	return &dockerClientMock{
		ContainersData: []types.Container{},
		ServicesData:   []swarm.Service{},
		ConfigsData:    []swarm.Config{},
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
