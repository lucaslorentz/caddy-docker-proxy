package generator

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
)

func TestServices_TemplateData(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s"):                           "{{.Spec.Name}}.testdomain.com",
						fmtLabel("%s.reverse_proxy"):             "{{.Spec.Name}}:5000",
						fmtLabel("%s.reverse_proxy.health_path"): "/health",
						fmtLabel("%s.gzip"):                      "",
						fmtLabel("%s.basicauth"):                 "/ user password",
						fmtLabel("%s.tls.dns"):                   "route53",
						fmtLabel("%s.rewrite_0"):                 "/path1 /path2",
						fmtLabel("%s.rewrite_1"):                 "/path3 /path4",
						fmtLabel("%s.limits.header"):             "100kb",
						fmtLabel("%s.limits.body_0"):             "/path1 2mb",
						fmtLabel("%s.limits.body_1"):             "/path2 4mb",
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

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	basicauth / user password\n" +
		"	gzip\n" +
		"	limits {\n" +
		"		body /path1 2mb\n" +
		"		body /path2 4mb\n" +
		"		header 100kb\n" +
		"	}\n" +
		"	reverse_proxy service:5000 {\n" +
		"		health_path /health\n" +
		"	}\n" +
		"	rewrite /path1 /path2\n" +
		"	rewrite /path3 /path4\n" +
		"	tls {\n" +
		"		dns route53\n" +
		"	}\n" +
		"}\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, skipCaddyfileText)
}

func TestServices_DifferentNetwork(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		{
			ID: "SERVICE-ID",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s"):               "service.testdomain.com",
						fmtLabel("%s.reverse_proxy"): "{{upstreams}}",
					},
				},
			},
			Endpoint: swarm.Endpoint{
				VirtualIPs: []swarm.EndpointVirtualIP{
					{
						NetworkID: "other-network-id",
					},
				},
			},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy service\n" +
		"}\n"

	const expectedLogs = skipCaddyfileText +
		"[WARNING] Service SERVICE-ID and caddy are not in same network\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, expectedLogs)
}

func TestServices_DifferentNetworkSkipValidation(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		{
			ID: "SERVICE-ID",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s"):               "service.testdomain.com",
						fmtLabel("%s.reverse_proxy"): "{{upstreams}}",
					},
				},
			},
			Endpoint: swarm.Endpoint{
				VirtualIPs: []swarm.EndpointVirtualIP{
					{
						NetworkID: "other-network-id",
					},
				},
			},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy service\n" +
		"}\n"

	const expectedLogs = skipCaddyfileText

	testGeneration(t, dockerClient, false, false, expectedCaddyfile, expectedLogs)
}

func TestServices_SwarmDisabled(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		{
			ID: "SERVICE-ID",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s"):               "service.testdomain.com",
						fmtLabel("%s.reverse_proxy"): "{{upstreams 5000}}",
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
	dockerClient.InfoData = types.Info{
		Swarm: swarm.Info{
			LocalNodeState: swarm.LocalNodeStateInactive,
		},
	}

	const expectedCaddyfile = "# Empty caddyfile"

	const expectedLogs = skipCaddyfileText +
		"[INFO] Skipping services because swarm is not available\n" +
		"[INFO] Skipping configs because swarm is not available\n"

	testGeneration(t, dockerClient, false, true, expectedCaddyfile, expectedLogs)
}

func TestServiceTasks_Empty(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		{
			ID: "SERVICEID",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s"):               "service.testdomain.com",
						fmtLabel("%s.reverse_proxy"): "{{upstreams 5000}}",
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

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy\n" +
		"}\n"

	const expectedLogs = skipCaddyfileText +
		"[WARNING] Service SERVICEID doesn't have any task in running state\n"

	testGeneration(t, dockerClient, true, true, expectedCaddyfile, expectedLogs)
}

func TestServiceTasks_NotRunning(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		{
			ID: "SERVICEID",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s"):               "service.testdomain.com",
						fmtLabel("%s.reverse_proxy"): "{{upstreams 5000}}",
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
	dockerClient.TasksData = []swarm.Task{
		{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				{
					Network: swarm.Network{
						ID: caddyNetworkID,
					},
					Addresses: []string{"10.0.0.1/24"},
				},
			},
			DesiredState: swarm.TaskStateShutdown,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		},
		{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				{
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

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy\n" +
		"}\n"

	const expectedLogs = skipCaddyfileText +
		"[WARNING] Service SERVICEID doesn't have any task in running state\n"

	testGeneration(t, dockerClient, true, true, expectedCaddyfile, expectedLogs)
}

func TestServiceTasks_DifferentNetwork(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		{
			ID: "SERVICEID",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s"):               "service.testdomain.com",
						fmtLabel("%s.reverse_proxy"): "{{upstreams 5000}}",
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
	dockerClient.TasksData = []swarm.Task{
		{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				{
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
		"	reverse_proxy\n" +
		"}\n"

	const expectedLogs = skipCaddyfileText +
		"[WARNING] Service SERVICEID and caddy are not in same network\n"

	testGeneration(t, dockerClient, true, true, expectedCaddyfile, expectedLogs)
}

func TestServiceTasks_DifferentNetworkSkipValidation(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		{
			ID: "SERVICEID",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s"):               "service.testdomain.com",
						fmtLabel("%s.reverse_proxy"): "{{upstreams 5000}}",
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
	dockerClient.TasksData = []swarm.Task{
		{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				{
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
		"	reverse_proxy 10.0.0.1:5000\n" +
		"}\n"

	testGeneration(t, dockerClient, true, false, expectedCaddyfile, skipCaddyfileText)
}

func TestServiceTasks_Running(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		{
			ID: "SERVICEID",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s"):               "service.testdomain.com",
						fmtLabel("%s.reverse_proxy"): "{{upstreams 5000}}",
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
	dockerClient.TasksData = []swarm.Task{
		{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				{
					Network: swarm.Network{
						ID: caddyNetworkID,
					},
					Addresses: []string{"10.0.0.1/24"},
				},
			},
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		},
		{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				{
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
		"	reverse_proxy 10.0.0.1:5000 10.0.0.2:5000\n" +
		"}\n"

	testGeneration(t, dockerClient, true, true, expectedCaddyfile, skipCaddyfileText)
}
