package generator

import (
	"bufio"
	"bytes"
	"errors"
	"testing"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/docker"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestServices_TemplateData(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						fmtLabel("%s"):                          "{{.Spec.Name}}.testdomain.com",
						fmtLabel("%s.reverse_proxy"):            "{{.Spec.Name}}:5000",
						fmtLabel("%s.reverse_proxy.health_uri"): "/health",
						fmtLabel("%s.gzip"):                     "",
						fmtLabel("%s.basicauth"):                "/ user password",
						fmtLabel("%s.tls.dns"):                  "route53",
						fmtLabel("%s.rewrite_0"):                "/path1 /path2",
						fmtLabel("%s.rewrite_1"):                "/path3 /path4",
						fmtLabel("%s.limits.header"):            "100kb",
						fmtLabel("%s.limits.body_0"):            "/path1 2mb",
						fmtLabel("%s.limits.body_1"):            "/path2 4mb",
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
		"		health_uri /health\n" +
		"	}\n" +
		"	rewrite /path1 /path2\n" +
		"	rewrite /path3 /path4\n" +
		"	tls {\n" +
		"		dns route53\n" +
		"	}\n" +
		"}\n"

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
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

	const expectedLogs = commonLogs +
		`WARN	Service is not in same network as caddy	{"service": "service", "serviceId": "SERVICE-ID"}` + newLine

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
}

func TestServices_ManualIngressNetwork(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.NetworksData = []network.Summary{
		{
			ID:   "other-network-id",
			Name: "other-network-name",
		},
	}
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

	const expectedLogs = swarmIsAvailableLog

	testGeneration(t, dockerClient, func(options *config.Options) {
		options.IngressNetworks = []string{"other-network-name"}
	}, expectedCaddyfile, expectedLogs)
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
	dockerClient.InfoData = system.Info{
		Swarm: swarm.Info{
			LocalNodeState: swarm.LocalNodeStateInactive,
		},
	}

	const expectedCaddyfile = "# Empty caddyfile"

	const expectedLogs = swarmIsDisabledLog

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
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

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, func(options *config.Options) {
		options.ProxyServiceTasks = true
	}, expectedCaddyfile, expectedLogs)
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

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, func(options *config.Options) {
		options.ProxyServiceTasks = true
	}, expectedCaddyfile, expectedLogs)
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

	const expectedLogs = commonLogs +
		`WARN	Service is not in same network as caddy	{"service": "service", "serviceId": "SERVICEID"}` + newLine

	testGeneration(t, dockerClient, func(options *config.Options) {
		options.ProxyServiceTasks = true
	}, expectedCaddyfile, expectedLogs)
}

func TestServiceTasks_ManualIngressNetwork(t *testing.T) {
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
	dockerClient.NetworksData = []network.Summary{
		{
			ID:   "other-network-id",
			Name: "other-network-name",
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

	const expectedLogs = swarmIsAvailableLog

	testGeneration(t, dockerClient, func(options *config.Options) {
		options.ProxyServiceTasks = true
		options.IngressNetworks = []string{"other-network-name"}
	}, expectedCaddyfile, expectedLogs)
}

func TestServiceTasks_OverrideIngressNetwork(t *testing.T) {
	dockerClient := createBasicDockerClientMock()
	dockerClient.ServicesData = []swarm.Service{
		{
			ID: "SERVICEID",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "service",
					Labels: map[string]string{
						"caddy_ingress_network":      "another-network",
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
	dockerClient.NetworksData = []network.Summary{
		{
			ID:   "other-network-id",
			Name: "other-network-name",
		},
		{
			ID:   "another-network-id",
			Name: "another-network-name",
		},
	}
	dockerClient.TasksData = []swarm.Task{
		{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				{
					Network: swarm.Network{
						ID: "other-network-id",
						Spec: swarm.NetworkSpec{
							Annotations: swarm.Annotations{
								Name: "other-network",
							},
						},
					},
					Addresses: []string{"10.0.0.1/24"},
				},
				{
					Network: swarm.Network{
						ID: "another-network-id",
						Spec: swarm.NetworkSpec{
							Annotations: swarm.Annotations{
								Name: "another-network",
							},
						},
					},
					Addresses: []string{"10.0.0.2/24"},
				},
			},
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		},
	}

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy 10.0.0.2:5000\n" +
		"}\n"

	const expectedLogs = swarmIsAvailableLog

	testGeneration(t, dockerClient, func(options *config.Options) {
		options.ProxyServiceTasks = true
		options.IngressNetworks = []string{"other-network-name"}
	}, expectedCaddyfile, expectedLogs)
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

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, func(options *config.Options) {
		options.ProxyServiceTasks = true
	}, expectedCaddyfile, expectedLogs)
}

// TestServiceTasks_OneClientTaskListError verifies that when running in
// multi-socket mode and one of the docker clients returns an error from
// TaskList (e.g. a Swarm worker responding "This node is not a swarm
// manager"), the generator continues to collect tasks from the remaining
// healthy clients instead of dropping the service's upstreams entirely.
//
// Regression test for #801 / fix for the pre-existing fail-fast in
// getServiceTasksIps that was carried over from the single-client era in
// PR #303 without adapting the error semantics for the multi-client loop.
func TestServiceTasks_OneClientTaskListError(t *testing.T) {
	managerClient := createBasicDockerClientMock()
	managerClient.ServicesData = []swarm.Service{
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
					{NetworkID: caddyNetworkID},
				},
			},
		},
	}
	managerClient.TasksData = []swarm.Task{
		{
			ServiceID: "SERVICEID",
			NetworksAttachments: []swarm.NetworkAttachment{
				{
					Network:   swarm.Network{ID: caddyNetworkID},
					Addresses: []string{"10.0.0.1/24"},
				},
			},
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		},
	}

	// Second client behaves like a Swarm worker: TaskList returns the
	// "not a swarm manager" error that real workers return.
	workerClient := createBasicDockerClientMock()
	workerClient.TaskListErr = errors.New("Error response from daemon: This node is not a swarm manager.")

	options := &config.Options{
		LabelPrefix:       DefaultLabelPrefix,
		ProxyServiceTasks: true,
	}

	dockerUtils := createDockerUtilsMock()
	generator := CreateGenerator(
		[]docker.Client{managerClient, workerClient},
		dockerUtils,
		options,
	)

	var logsBuffer bytes.Buffer
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = ""
	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	writer := bufio.NewWriter(&logsBuffer)
	logger := zap.New(zapcore.NewCore(encoder, zapcore.AddSync(writer), zapcore.InfoLevel))

	caddyfileBytes, _ := generator.GenerateCaddyfile(logger)
	writer.Flush()

	const expectedCaddyfile = "service.testdomain.com {\n" +
		"	reverse_proxy 10.0.0.1:5000\n" +
		"}\n"

	assert.Equal(t, expectedCaddyfile, string(caddyfileBytes),
		"manager client's task IP must still reach the Caddyfile when a peer client's TaskList errors")
}
