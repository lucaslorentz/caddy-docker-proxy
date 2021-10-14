package generator

import (
	"bufio"
	"bytes"
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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var caddyContainerID = "container-id"
var caddyNetworkID = "network-id"

const newLine = "\n"
const containerIdLog = `INFO	Caddy ContainerID	{"ID": "container-id"}` + newLine
const ingressNetworksMapLog = `INFO	IngressNetworksMap	{"ingres": "map[network-id:true]"}` + newLine
const otherIngressNetworksMapLog = `INFO	IngressNetworksMap	{"ingres": "map[other-network-id:true]"}` + newLine
const swarmIsAvailableLog = `INFO	Swarm is available	{"new": true}` + newLine
const swarmIsDisabledLog = `INFO	Swarm is available	{"new": false}` + newLine
const skipCaddyfileLog = "INFO	Skipping default Caddyfile because no path is set" + newLine
const commonLogs = containerIdLog + ingressNetworksMapLog + swarmIsAvailableLog

func init() {
	log.SetOutput(ioutil.Discard)
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

	const expectedLogs = commonLogs + skipCaddyfileLog

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

	const expectedLogs = commonLogs + skipCaddyfileLog

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
		ContainersData: []types.Container{},
		ServicesData:   []swarm.Service{},
		ConfigsData:    []swarm.Config{},
		TasksData:      []swarm.Task{},
		NetworksData:   []types.NetworkResource{},
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
