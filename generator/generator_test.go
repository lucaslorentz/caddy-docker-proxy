package generator

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/netip"
	"testing"

	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/docker"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/system"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var caddyContainerID = "container-id"
var caddyNetworkID = "network-id"
var caddyNetworkName = "network-name"

const newLine = "\n"
const swarmIsAvailableLog = `INFO	Swarm is available	{"new": true}` + newLine
const swarmIsDisabledLog = `INFO	Swarm is available	{"new": false}` + newLine
const commonLogs = swarmIsAvailableLog

func init() {
	log.SetOutput(io.Discard)
}

func fmtLabel(s string) string {
	return fmt.Sprintf(s, DefaultLabelPrefix)
}

func TestMergeConfigContent(t *testing.T) {
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
					"{\n" +
						"	email test@example.com\n" +
						"}\n" +
						"example.com {\n" +
						"	reverse_proxy 127.0.0.1\n" +
						"}",
				),
			},
		},
	}
	dockerClient.ContainersData = []container.Summary{
		{
			Names: []string{
				"container-name",
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
				fmtLabel("%s"):                      "example.com",
				fmtLabel("%s.reverse_proxy"):        "{{upstreams}}",
				fmtLabel("%s_1.experimental_http3"): "",
			},
		},
	}

	const expectedCaddyfile = "{\n" +
		"	email test@example.com\n" +
		"	experimental_http3\n" +
		"}\n" +
		"example.com {\n" +
		"	reverse_proxy 127.0.0.1 172.17.0.2\n" +
		"}\n"

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
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

	const expectedLogs = commonLogs

	testGeneration(t, dockerClient, nil, expectedCaddyfile, expectedLogs)
}

func testGeneration(
	t *testing.T,
	dockerClient docker.Client,
	customizeOptions func(*config.Options),
	expectedCaddyfile string,
	expectedLogs string,
) {
	dockerUtils := createDockerUtilsMock()

	options := &config.Options{
		LabelPrefix: DefaultLabelPrefix,
	}

	if customizeOptions != nil {
		customizeOptions(options)
	}

	generator := CreateGenerator([]docker.Client{dockerClient}, dockerUtils, options)

	var logsBuffer bytes.Buffer
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = ""
	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	writer := bufio.NewWriter(&logsBuffer)
	logger := zap.New(zapcore.NewCore(encoder, zapcore.AddSync(writer), zapcore.InfoLevel))

	caddyfileBytes, _ := generator.GenerateCaddyfile(logger)
	writer.Flush()
	assert.Equal(t, expectedCaddyfile, string(caddyfileBytes))
	assert.Equal(t, expectedLogs, logsBuffer.String())
}

func createBasicDockerClientMock() *docker.ClientMock {
	return &docker.ClientMock{
		ContainersData: []container.Summary{},
		ServicesData:   []swarm.Service{},
		ConfigsData:    []swarm.Config{},
		TasksData:      []swarm.Task{},
		NetworksData:   []network.Summary{},
		InfoData: system.Info{
			Swarm: swarm.Info{
				LocalNodeState: swarm.LocalNodeStateActive,
			},
		},
		ContainerInspectData: map[string]container.InspectResponse{
			caddyContainerID: {
				NetworkSettings: &container.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{
						"overlay": {
							NetworkID: caddyNetworkID,
						},
					},
				},
			},
		},
		NetworkInspectData: map[string]network.Inspect{
			caddyNetworkID: networkInspect(caddyNetworkID, caddyNetworkName),
		},
	}
}

func prefixes(values ...string) []netip.Prefix {
	result := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		result = append(result, netip.MustParsePrefix(value))
	}
	return result
}

func networkSummary(id string, name string) network.Summary {
	return network.Summary{
		Network: network.Network{
			ID:   id,
			Name: name,
		},
	}
}

func networkInspect(id string, name string) network.Inspect {
	return network.Inspect{
		Network: network.Network{
			ID:   id,
			Name: name,
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
