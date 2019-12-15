package plugin

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
)

func TestContainers_Templates(t *testing.T) {
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

	const expectedCaddyfile = "container-name.testdomain.com {\n" +
		"  proxy / 172.17.0.2:5000/api\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestContainers_PicksRightNetwork(t *testing.T) {
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

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"  proxy / 172.17.0.2\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestContainers_MinimumBasicLabels(t *testing.T) {
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

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"  proxy / 172.17.0.2\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestContainers_AllBasicLabels(t *testing.T) {
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

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"  proxy / https://172.17.0.2:5000/api\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestContainers_DifferentNetwork(t *testing.T) {
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

	const expectedCaddyfile = ""

	const expectedLogs = skipCaddyfileText +
		"[ERROR] Container CONTAINER-ID and caddy are not in same network\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, expectedLogs)
}

func TestContainers_DifferentNetworkSkipValidation(t *testing.T) {
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

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"  proxy / 10.0.0.1\n" +
		"}\n"

	const expectedLogs = skipCaddyfileText

	testGeneration(t, dockerClient, false, false, expectedCaddyfile, expectedLogs)
}

func TestContainers_MultipleConfigs(t *testing.T) {
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

	const expectedCaddyfile = "service1.testdomain.com {\n" +
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

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestContainers_Replicas(t *testing.T) {
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

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"  proxy / 172.17.0.2 172.17.0.3\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestContainers_DoNotMergeProxiesWithDifferentLabelKey(t *testing.T) {
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

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"  proxy /a service-a\n" +
		"  proxy /b service-b\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestContainers_WithSnippets(t *testing.T) {
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

	const expectedCaddyfile = "(mysnippet-1) {\n" +
		"  tls off\n" +
		"}\n" +
		"(mysnippet-2) {\n" +
		"  tls off\n" +
		"}\n" +
		"service.testdomain.com {\n" +
		"  import mysnippet-1\n" +
		"  proxy / 172.17.0.3\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}
