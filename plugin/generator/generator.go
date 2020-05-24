package generator

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"regexp"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/lucaslorentz/caddy-docker-proxy/plugin/v2/caddyfile"
	"github.com/lucaslorentz/caddy-docker-proxy/plugin/v2/config"
	"github.com/lucaslorentz/caddy-docker-proxy/plugin/v2/docker"
)

// DefaultLabelPrefix for caddy labels in docker
const DefaultLabelPrefix = "caddy"

const swarmAvailabilityCacheInterval = 1 * time.Minute

// CaddyfileGenerator generates caddyfile from docker configuration
type CaddyfileGenerator struct {
	options              *config.Options
	labelRegex           *regexp.Regexp
	dockerClient         docker.Client
	dockerUtils          docker.Utils
	caddyNetworks        map[string]bool
	swarmIsAvailable     bool
	swarmIsAvailableTime time.Time
}

// CreateGenerator creates a new generator
func CreateGenerator(dockerClient docker.Client, dockerUtils docker.Utils, options *config.Options) *CaddyfileGenerator {
	var labelRegexString = fmt.Sprintf("^%s(_\\d+)?(\\.|$)", options.LabelPrefix)

	return &CaddyfileGenerator{
		options:      options,
		labelRegex:   regexp.MustCompile(labelRegexString),
		dockerClient: dockerClient,
		dockerUtils:  dockerUtils,
	}
}

// GenerateCaddyfile generates a caddy file config from docker metadata
func (g *CaddyfileGenerator) GenerateCaddyfile() ([]byte, string, []string) {
	var caddyfileBuffer bytes.Buffer
	var logsBuffer bytes.Buffer

	if g.options.ValidateNetwork && g.caddyNetworks == nil {
		networks, err := g.getCaddyNetworks()
		if err == nil {
			g.caddyNetworks = map[string]bool{}
			for _, network := range networks {
				g.caddyNetworks[network] = true
			}
		} else {
			logsBuffer.WriteString(fmt.Sprintf("[ERROR] %v\n", err.Error()))
		}
	}

	if time.Since(g.swarmIsAvailableTime) > swarmAvailabilityCacheInterval {
		g.checkSwarmAvailability(time.Time.IsZero(g.swarmIsAvailableTime))
		g.swarmIsAvailableTime = time.Now()
	}

	caddyfileBlock := caddyfile.CreateBlock()

	if g.options.CaddyfilePath != "" {
		dat, err := ioutil.ReadFile(g.options.CaddyfilePath)

		if err == nil {
			_, err = caddyfileBuffer.Write(dat)
		}

		if err != nil {
			logsBuffer.WriteString(fmt.Sprintf("[ERROR] %v\n", err.Error()))
		}
	} else {
		logsBuffer.WriteString("[INFO] Skipping default Caddyfile because no path is set\n")
	}

	controlledServers := []string{}

	containers, err := g.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err == nil {
		for _, container := range containers {
			if _, isControlledServer := container.Labels[g.options.ControlledServersLabel]; isControlledServer {
				ips, err := g.getContainerIPAddresses(&container, &logsBuffer)
				if err != nil {
					logsBuffer.WriteString(fmt.Sprintf("[ERROR] %v\n", err.Error()))
				} else {
					for _, ip := range ips {
						if g.options.ControllerNetwork == nil || g.options.ControllerNetwork.Contains(net.ParseIP(ip)) {
							controlledServers = append(controlledServers, ip)
						}
					}
				}
			}

			containerCaddyfile, err := g.getContainerCaddyfile(&container, &logsBuffer)
			if err == nil {
				caddyfileBlock.Merge(containerCaddyfile)
			} else {
				logsBuffer.WriteString(fmt.Sprintf("[ERROR] %v\n", err.Error()))
			}
		}
	} else {
		logsBuffer.WriteString(fmt.Sprintf("[ERROR] %v\n", err.Error()))
	}

	if g.swarmIsAvailable {
		services, err := g.dockerClient.ServiceList(context.Background(), types.ServiceListOptions{})
		if err == nil {
			for _, service := range services {
				if _, isControlledServer := service.Spec.Labels[g.options.ControlledServersLabel]; isControlledServer {
					ips, err := g.getServiceTasksIps(&service, &logsBuffer)
					if err != nil {
						logsBuffer.WriteString(fmt.Sprintf("[ERROR] %v\n", err.Error()))
					} else {
						for _, ip := range ips {
							if g.options.ControllerNetwork == nil || g.options.ControllerNetwork.Contains(net.ParseIP(ip)) {
								controlledServers = append(controlledServers, ip)
							}
						}
					}
				}

				serviceCaddyfile, err := g.getServiceCaddyfile(&service, &logsBuffer)
				if err == nil {
					caddyfileBlock.Merge(serviceCaddyfile)
				} else {
					logsBuffer.WriteString(fmt.Sprintf("[ERROR] %v\n", err.Error()))
				}
			}
		} else {
			logsBuffer.WriteString(fmt.Sprintf("[ERROR] %v\n", err.Error()))
		}
	} else {
		logsBuffer.WriteString("[INFO] Skipping services because swarm is not available\n")
	}

	// Write global blocks first
	for _, directive := range caddyfileBlock.Children {
		if directive.IsGlobalBlock() {
			directive.Write(&caddyfileBuffer, 0)
			caddyfileBlock.Remove(directive)
		}
	}

	// Write swarm configs
	if g.swarmIsAvailable {
		configs, err := g.dockerClient.ConfigList(context.Background(), types.ConfigListOptions{})
		if err == nil {
			for _, config := range configs {
				if _, hasLabel := config.Spec.Labels[g.options.LabelPrefix]; hasLabel {
					fullConfig, _, err := g.dockerClient.ConfigInspectWithRaw(context.Background(), config.ID)
					if err == nil {
						caddyfileBuffer.Write(fullConfig.Spec.Data)
						caddyfileBuffer.WriteRune('\n')
					} else {
						logsBuffer.WriteString(fmt.Sprintf("[ERROR] %v\n", err.Error()))
					}
				}
			}
		} else {
			logsBuffer.WriteString(fmt.Sprintf("[ERROR] %v\n", err.Error()))
		}
	} else {
		logsBuffer.WriteString("[INFO] Skipping configs because swarm is not available\n")
	}

	// Write remaining blocks
	caddyfileBlock.Write(&caddyfileBuffer, 0)

	caddyfileContent := caddyfileBuffer.Bytes()

	if g.options.ProcessCaddyfile {
		processCaddyfileContent, processLogs := caddyfile.Process(caddyfileContent)
		caddyfileContent = processCaddyfileContent
		logsBuffer.Write(processLogs)
	}

	if len(caddyfileContent) == 0 {
		caddyfileContent = []byte("# Empty caddyfile")
	}

	if g.options.Mode&config.Server == config.Server {
		controlledServers = append(controlledServers, "localhost")
	}

	return caddyfileContent, logsBuffer.String(), controlledServers
}

func (g *CaddyfileGenerator) checkSwarmAvailability(isFirstCheck bool) {
	info, err := g.dockerClient.Info(context.Background())
	if err == nil {
		newSwarmIsAvailable := info.Swarm.LocalNodeState == swarm.LocalNodeStateActive
		if isFirstCheck || newSwarmIsAvailable != g.swarmIsAvailable {
			log.Printf("[INFO] Swarm is available: %v\n", newSwarmIsAvailable)
		}
		g.swarmIsAvailable = newSwarmIsAvailable
	} else {
		log.Printf("[ERROR] Swarm availability check failed: %v\n", err.Error())
		g.swarmIsAvailable = false
	}
}

func (g *CaddyfileGenerator) getCaddyNetworks() ([]string, error) {
	containerID, err := g.dockerUtils.GetCurrentContainerID()
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Caddy ContainerID: %v\n", containerID)
	container, err := g.dockerClient.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return nil, err
	}

	var networks []string
	for _, network := range container.NetworkSettings.Networks {
		networkInfo, err := g.dockerClient.NetworkInspect(context.Background(), network.NetworkID, types.NetworkInspectOptions{})
		if err != nil {
			return nil, err
		}
		if !networkInfo.Ingress {
			networks = append(networks, network.NetworkID)
		}
	}
	log.Printf("[INFO] Caddy Networks: %v\n", networks)

	return networks, nil
}

func (g *CaddyfileGenerator) filterLabels(labels map[string]string) map[string]string {
	filteredLabels := map[string]string{}
	for label, value := range labels {
		if g.labelRegex.MatchString(label) {
			filteredLabels[label] = value
		}
	}
	return filteredLabels
}
