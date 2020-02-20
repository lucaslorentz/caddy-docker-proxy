package plugin

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
)

func TestServices_Templates(t *testing.T) {
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

	const expectedCaddyfile = "service.testdomain.com {\n" +
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

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestServices_MinimumBasicLabels(t *testing.T) {
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

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"  proxy / service\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestServices_AllBasicLabels(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s.address"):        "service.testdomain.com",
						fmtLabel("%s.sourcepath"):     "/source",
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

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"  proxy /source https://service:5000/api\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestServices_MultipleConfigs(t *testing.T) {
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

	const expectedCaddyfile = "service1.testdomain.com {\n" +
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

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestServices_MultipleAddresses(t *testing.T) {
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

	const expectedCaddyfile = "a.testdomain.com b.testdomain.com {\n" +
		"  proxy / service\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
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

	const expectedCaddyfile = "testdomain.com {\n" +
		"  proxy / something\n" +
		"  proxy /api external-api\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestServices_DifferentNetwork(t *testing.T) {
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

	const expectedCaddyfile = ""

	const expectedLogs = skipCaddyfileText +
		"[ERROR] Service SERVICE-ID and caddy are not in same network\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, expectedLogs)
}

func TestServices_DifferentNetworkSkipValidation(t *testing.T) {
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

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"  proxy / service\n" +
		"}\n"

	const expectedLogs = skipCaddyfileText

	testGeneration(t, dockerClient, false, false, expectedCaddyfile, expectedLogs)
}

func TestServices_SwarmDisabled(t *testing.T) {
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

	const expectedCaddyfile = ""

	const expectedLogs = skipCaddyfileText +
		"[INFO] Skipping services because swarm is not available\n" +
		"[INFO] Skipping configs because swarm is not available\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, expectedLogs)
}

func TestServiceTasks_Empty(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			ID: "SERVICEID",
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

	const expectedCaddyfile = ""

	const expectedLogs = skipCaddyfileText +
		"[ERROR] Service SERVICEID doesn't have any task in running state\n"

	testGeneration(t, dockerClient, true, true, expectedCaddyfile, expectedLogs)
}

func TestServiceTasks_NotRunning(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			ID: "SERVICEID",
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
	dockerClient.TasksData = []swarm.Task{
		swarm.Task{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				swarm.NetworkAttachment{
					Network: swarm.Network{
						ID: caddyNetworkID,
					},
					Addresses: []string{"10.0.0.1/24"},
				},
			},
			DesiredState: swarm.TaskStateShutdown,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		},
		swarm.Task{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				swarm.NetworkAttachment{
					Network: swarm.Network{
						ID: caddyNetworkID,
					},
					Addresses: []string{"10.0.0.2/24"},
				},
			},
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateShutdown},
		},
	}

	const expectedCaddyfile = ""

	const expectedLogs = skipCaddyfileText +
		"[ERROR] Service SERVICEID doesn't have any task in running state\n"

	testGeneration(t, dockerClient, true, true, expectedCaddyfile, expectedLogs)
}

func TestServiceTasks_DifferentNetwork(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			ID: "SERVICEID",
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
	dockerClient.TasksData = []swarm.Task{
		swarm.Task{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				swarm.NetworkAttachment{
					Network: swarm.Network{
						ID: "other-network-id",
					},
					Addresses: []string{"10.0.0.1/24"},
				},
			},
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		},
	}

	const expectedCaddyfile = ""

	const expectedLogs = skipCaddyfileText +
		"[ERROR] Service SERVICEID and caddy are not in same network\n"

	testGeneration(t, dockerClient, true, true, expectedCaddyfile, expectedLogs)
}

func TestServiceTasks_DifferentNetworkSkipValidation(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			ID: "SERVICEID",
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
	dockerClient.TasksData = []swarm.Task{
		swarm.Task{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				swarm.NetworkAttachment{
					Network: swarm.Network{
						ID: "other-network-id",
					},
					Addresses: []string{"10.0.0.1/24"},
				},
			},
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"  proxy / 10.0.0.1:5000\n" +
		"}\n"

	testGeneration(t, dockerClient, true, false, expectedCaddyfile, skipCaddyfileText)
}

func TestServiceTasks_Running(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		swarm.Service{
			ID: "SERVICEID",
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
	dockerClient.TasksData = []swarm.Task{
		swarm.Task{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				swarm.NetworkAttachment{
					Network: swarm.Network{
						ID: caddyNetworkID,
					},
					Addresses: []string{"10.0.0.1/24"},
				},
			},
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		},
		swarm.Task{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				swarm.NetworkAttachment{
					Network: swarm.Network{
						ID: caddyNetworkID,
					},
					Addresses: []string{"10.0.0.2/24"},
				},
			},
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"  proxy / 10.0.0.1:5000 10.0.0.2:5000\n" +
		"}\n"

	testGeneration(t, dockerClient, true, true, expectedCaddyfile, skipCaddyfileText)
}
