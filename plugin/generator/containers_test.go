package generator

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
)

func TestContainers_TemplateData(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		{
			Names: []string{
				"container-name",
			},
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: "172.17.0.2",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):               "{{index .Names 0}}.testdomain.com",
				fmtLabel("%s.reverse_proxy"): "{{(index .NetworkSettings.Networks \"caddy-network\").IPAddress}}:5000/api",
			},
		},
	}

	const expectedCaddyfile = "container-name.testdomain.com {\n" +
		"	reverse_proxy 172.17.0.2:5000/api\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestContainers_PicksRightNetwork(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"other-network": {
						IPAddress: "10.0.0.1",
						NetworkID: "other-network-id",
					},
					"caddy-network": {
						IPAddress: "172.17.0.2",
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

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestContainers_DifferentNetwork(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		{
			ID: "CONTAINER-ID",
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"other-network": {
						IPAddress: "10.0.0.1",
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

	const expectedLogs = skipCaddyfileText +
		"[WARNING] Container CONTAINER-ID and caddy are not in same network\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, expectedLogs)
}

func TestContainers_DifferentNetworkSkipValidation(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		{
			ID: "CONTAINER-ID",
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"other-network": {
						IPAddress: "10.0.0.1",
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

	const expectedLogs = skipCaddyfileText

	testGeneration(t, dockerClient, false, false, expectedCaddyfile, expectedLogs)
}

func TestContainers_Replicas(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: "172.17.0.2",
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
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: "172.17.0.3",
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

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestContainers_DoNotMergeDifferentProxies(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: "172.17.0.2",
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
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: "172.17.0.3",
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

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestContainers_ComplexMerge(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: "172.17.0.2",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):                                 "service.testdomain.com",
				fmtLabel("%s.route"):                           "/a/*",
				fmtLabel("%s.route.0_uri"):                     "strip_prefix /a",
				fmtLabel("%s.route.reverse_proxy"):             "{{upstreams}}",
				fmtLabel("%s.route.reverse_proxy.health_path"): "/health",
				fmtLabel("%s.redir"):                           "/a /a1",
				fmtLabel("%s.tls"):                             "internal",
			},
		},
		{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: "172.17.0.3",
						NetworkID: caddyNetworkID,
					},
				},
			},
			Labels: map[string]string{
				fmtLabel("%s"):                                 "service.testdomain.com",
				fmtLabel("%s.route"):                           "/b/*",
				fmtLabel("%s.route.0_uri"):                     "strip_prefix /b",
				fmtLabel("%s.route.reverse_proxy"):             "{{upstreams}}",
				fmtLabel("%s.route.reverse_proxy.health_path"): "/health",
				fmtLabel("%s.redir"):                           "/b /b1",
				fmtLabel("%s.tls"):                             "internal",
			},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	redir /a /a1\n" +
		"	redir /b /b1\n" +
		"	route /a/* {\n" +
		"		uri strip_prefix /a\n" +
		"		reverse_proxy 172.17.0.2 {\n" +
		"			health_path /health\n" +
		"		}\n" +
		"	}\n" +
		"	route /b/* {\n" +
		"		uri strip_prefix /b\n" +
		"		reverse_proxy 172.17.0.3 {\n" +
		"			health_path /health\n" +
		"		}\n" +
		"	}\n" +
		"	tls internal\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestContainers_WithSnippets(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ContainersData = []types.Container{
		{
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: "172.17.0.3",
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
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"caddy-network": {
						IPAddress: "172.17.0.2",
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

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}
