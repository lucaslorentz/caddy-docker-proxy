package generator

import (
	"net/netip"
	"testing"

	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
)

func TestContainers_TemplateData(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []container.Summary{
		{
			Names: []string{
				"/container-name",
			},
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: netip.MustParseAddr("172.17.0.2"),
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):               "{{trimPrefix \"/\" (index .Names 0)}}.testdomain.com",
				fmtLabel("%s.reverse_proxy"): "{{(index .NetworkSettings.Networks \"caddy-network\").IPAddress}}:5000/api",
			},
		},
	}

	const expectedCaddyfile = "container-name.testdomain.com {\n" +
		"	reverse_proxy 172.17.0.2:5000/api\n" +
		"}\n"

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
}

func TestContainers_PicksRightNetwork(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []container.Summary{
		{
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"other-network": {
						IPAddress: netip.MustParseAddr("10.0.0.1"),
						NetworkID: "other-network-id",
					},
					"caddy-network": {
						IPAddress: netip.MustParseAddr("172.17.0.2"),
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):               "service.testdomain.com",
				fmtLabel("%s.reverse_proxy"): "{{upstreams}}",
			},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy 172.17.0.2\n" +
		"}\n"

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
}

func TestContainers_DifferentNetwork(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []container.Summary{
		{
			ID:    "CONTAINER-ID",
			Names: []string{"/lonely-container"},
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"other-network": {
						IPAddress: netip.MustParseAddr("10.0.0.1"),
						NetworkID: "other-network-id",
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):               "service.testdomain.com",
				fmtLabel("%s.reverse_proxy"): "{{upstreams}}",
			},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy\n" +
		"}\n"

	const expectedLogs = commonLogs +
		`WARN	Container is not in same network as caddy	{"container": "lonely-container", "container networks": ["other-network"], "ingress networks": []}` + newLine

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
}

// A stopped container surfaced by CADDY_DOCKER_SCAN_STOPPED_CONTAINERS still
// reports its ingress network but has released its runtime IP address. It is in
// the same network as caddy, so it must not emit the "not in same network"
// warning (matched here by network ID).
func TestContainers_StoppedContainerInIngressNetworkDoesNotWarn(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []container.Summary{
		{
			ID:    "CONTAINER-ID",
			Names: []string{"/stopped-container"},
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						// No IPAddress: a stopped container has no runtime IP.
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):               "service.testdomain.com",
				fmtLabel("%s.reverse_proxy"): "{{upstreams}}",
			},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy\n" +
		"}\n"

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
}

// Same as above, but the stopped container reports no runtime NetworkID, so the
// ingress network is matched by name instead. Still no warning.
func TestContainers_StoppedContainerMatchesIngressByName(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []container.Summary{
		{
			ID:    "CONTAINER-ID",
			Names: []string{"/stopped-container"},
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					// Keyed by the ingress network name, with neither a runtime
					// NetworkID nor an IP address.
					caddyNetworkName: {},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):               "service.testdomain.com",
				fmtLabel("%s.reverse_proxy"): "{{upstreams}}",
			},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy\n" +
		"}\n"

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
}

func TestContainers_ManualIngressNetworks(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.NetworksData = []network.Summary{
		networkSummary("other-network-id", "other-network-name"),
	}
	dockerClient.ContainersData = []container.Summary{
		{
			ID: "CONTAINER-ID",
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"other-network": {
						IPAddress: netip.MustParseAddr("10.0.0.1"),
						NetworkID: "other-network-id",
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):               "service.testdomain.com",
				fmtLabel("%s.reverse_proxy"): "{{upstreams}}",
			},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy 10.0.0.1\n" +
		"}\n"

	const expectedLogs = swarmIsAvailableLog

	testGeneration(t, dockerClient, func(options *config.Options) {
		options.IngressNetworks = []string{"other-network-name"}
	}, expectedCaddyfile, expectedLogs)
}

func TestContainers_OverrideIngressNetworks(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.NetworksData = []network.Summary{
		networkSummary("other-network-id", "other-network-name"),
		networkSummary("another-network-id", "another-network-name"),
	}
	dockerClient.ContainersData = []container.Summary{
		{
			ID: "CONTAINER-ID",
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"other-network": {
						IPAddress: netip.MustParseAddr("10.0.0.1"),
						NetworkID: "other-network-id",
					},
					"another-network": {
						IPAddress: netip.MustParseAddr("10.0.0.2"),
						NetworkID: "other-network-id",
					},
				},
			},
			Labels: map[string]string{
				"caddy_ingress_network":      "another-network",
				fmtLabel("%s"):               "service.testdomain.com",
				fmtLabel("%s.reverse_proxy"): "{{upstreams}}",
			},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy 10.0.0.2\n" +
		"}\n"

	const expectedLogs = swarmIsAvailableLog

	testGeneration(t, dockerClient, func(options *config.Options) {
		options.IngressNetworks = []string{"other-network-name"}
	}, expectedCaddyfile, expectedLogs)
}

func TestContainers_Replicas(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []container.Summary{
		{
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: netip.MustParseAddr("172.17.0.2"),
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):               "service.testdomain.com",
				fmtLabel("%s.reverse_proxy"): "{{upstreams}}",
			},
		},
		{
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: netip.MustParseAddr("172.17.0.3"),
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):               "service.testdomain.com",
				fmtLabel("%s.reverse_proxy"): "{{upstreams}}",
			},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy 172.17.0.2 172.17.0.3\n" +
		"}\n"

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
}

func TestContainers_DoNotMergeDifferentProxies(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []container.Summary{
		{
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: netip.MustParseAddr("172.17.0.2"),
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):               "service.testdomain.com",
				fmtLabel("%s.reverse_proxy"): "/a/* {{upstreams}}",
			},
		},
		{
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: netip.MustParseAddr("172.17.0.3"),
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):               "service.testdomain.com",
				fmtLabel("%s.reverse_proxy"): "/b/* {{upstreams}}",
			},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy /a/* 172.17.0.2\n" +
		"	reverse_proxy /b/* 172.17.0.3\n" +
		"}\n"

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
}

func TestContainers_ComplexMerge(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []container.Summary{
		{
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: netip.MustParseAddr("172.17.0.2"),
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):                                "service.testdomain.com",
				fmtLabel("%s.route"):                          "/a/*",
				fmtLabel("%s.route.0_uri"):                    "strip_prefix /a",
				fmtLabel("%s.route.reverse_proxy"):            "{{upstreams}}",
				fmtLabel("%s.route.reverse_proxy.health_uri"): "/health",
				fmtLabel("%s.redir"):                          "/a /a1",
				fmtLabel("%s.tls"):                            "internal",
			},
		},
		{
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: netip.MustParseAddr("172.17.0.3"),
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):                                "service.testdomain.com",
				fmtLabel("%s.route"):                          "/b/*",
				fmtLabel("%s.route.0_uri"):                    "strip_prefix /b",
				fmtLabel("%s.route.reverse_proxy"):            "{{upstreams}}",
				fmtLabel("%s.route.reverse_proxy.health_uri"): "/health",
				fmtLabel("%s.redir"):                          "/b /b1",
				fmtLabel("%s.tls"):                            "internal",
			},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	redir /a /a1\n" +
		"	redir /b /b1\n" +
		"	route /a/* {\n" +
		"		uri strip_prefix /a\n" +
		"		reverse_proxy 172.17.0.2 {\n" +
		"			health_uri /health\n" +
		"		}\n" +
		"	}\n" +
		"	route /b/* {\n" +
		"		uri strip_prefix /b\n" +
		"		reverse_proxy 172.17.0.3 {\n" +
		"			health_uri /health\n" +
		"		}\n" +
		"	}\n" +
		"	tls internal\n" +
		"}\n"

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
}

func TestContainers_WithSnippets(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []container.Summary{
		{
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: netip.MustParseAddr("172.17.0.3"),
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):               "service.testdomain.com",
				fmtLabel("%s.reverse_proxy"): "{{upstreams}}",
				fmtLabel("%s.import"):        "mysnippet-1",
			},
		},
		{
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: netip.MustParseAddr("172.17.0.2"),
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s_1"):     "(mysnippet-1)",
				fmtLabel("%s_1.tls"): "internal",
				fmtLabel("%s_2"):     "(mysnippet-2)",
				fmtLabel("%s_2.tls"): "internal",
			},
		},
	}

	const expectedCaddyfile = "(mysnippet-1) {\n" +
		"	tls internal\n" +
		"}\n" +
		"(mysnippet-2) {\n" +
		"	tls internal\n" +
		"}\n" +
		"service.testdomain.com {\n" +
		"	import mysnippet-1\n" +
		"	reverse_proxy 172.17.0.3\n" +
		"}\n"

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
}
