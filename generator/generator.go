package generator

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"regexp"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/caddyfile"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/docker"

	"go.uber.org/zap"
)

// DefaultLabelPrefix for caddy labels in docker
const DefaultLabelPrefix = "caddy"

const swarmAvailabilityCacheInterval = 1 * time.Minute

// CaddyfileGenerator generates caddyfile from docker configuration
type CaddyfileGenerator struct {
	options              *config.Options
	labelRegex           *regexp.Regexp
	dockerClients        []docker.Client
	dockerUtils          docker.Utils
	ingressNetworks      *NetworkGroup
	controllerNetworks   *NetworkGroup
	swarmIsAvailable     []bool
	swarmIsAvailableTime time.Time
}

// CreateGenerator creates a new generator
func CreateGenerator(dockerClients []docker.Client, dockerUtils docker.Utils, options *config.Options) *CaddyfileGenerator {
	var labelRegexString = fmt.Sprintf("^%s(_\\d+)?(\\.|$)", options.LabelPrefix)

	return &CaddyfileGenerator{
		options:          options,
		labelRegex:       regexp.MustCompile(labelRegexString),
		dockerClients:    dockerClients,
		swarmIsAvailable: make([]bool, len(dockerClients)),
		dockerUtils:      dockerUtils,
	}
}

// GenerateCaddyfile generates a caddy file config from docker metadata
func (g *CaddyfileGenerator) GenerateCaddyfile(logger *zap.Logger) ([]byte, []string) {
	var caddyfileBuffer bytes.Buffer

	if g.ingressNetworks == nil {
		ingressNetworks, err := g.createNetworkGroup(logger, "ingress", g.options.IngressNetworks)
		if err == nil {
			g.ingressNetworks = ingressNetworks
		} else {
			logger.Error("Failed to get ingress networks", zap.Error(err))
		}
	}

	if g.controllerNetworks == nil {
		controllerNetworks, err := g.createNetworkGroup(logger, "controller", []string{g.options.ControllerNetwork})
		if err == nil {
			g.controllerNetworks = controllerNetworks
		} else {
			logger.Error("Failed to get controller networks", zap.Error(err))
		}
	}

	if time.Since(g.swarmIsAvailableTime) > swarmAvailabilityCacheInterval {
		g.checkSwarmAvailability(logger, time.Time.IsZero(g.swarmIsAvailableTime))
		g.swarmIsAvailableTime = time.Now()
	}

	caddyfileBlock := caddyfile.CreateContainer()
	controlledServers := []string{}

	// Add caddyfile from path
	if g.options.CaddyfilePath != "" {
		dat, err := ioutil.ReadFile(g.options.CaddyfilePath)
		if err != nil {
			logger.Error("Failed to read Caddyfile", zap.String("path", g.options.CaddyfilePath), zap.Error(err))
		} else {
			block, err := caddyfile.Unmarshal(dat)
			if err != nil {
				logger.Error("Failed to parse Caddyfile", zap.String("path", g.options.CaddyfilePath), zap.Error(err))
			} else {
				caddyfileBlock.Merge(block)
			}
		}
	} else {
		logger.Debug("Skipping default Caddyfile because no path is set")
	}

	for i, dockerClient := range g.dockerClients {

		// Add Caddyfile from swarm configs
		if g.swarmIsAvailable[i] {
			configs, err := dockerClient.ConfigList(context.Background(), types.ConfigListOptions{})
			if err == nil {
				for _, config := range configs {
					if _, hasLabel := config.Spec.Labels[g.options.LabelPrefix]; hasLabel {
						fullConfig, _, err := dockerClient.ConfigInspectWithRaw(context.Background(), config.ID)
						if err != nil {
							logger.Error("Failed to inspect Swarm Config", zap.String("config", config.Spec.Name), zap.Error(err))

						} else {
							block, err := caddyfile.Unmarshal(fullConfig.Spec.Data)
							if err != nil {
								logger.Error("Failed to parse Swarm Config caddyfile format", zap.String("config", config.Spec.Name), zap.Error(err))
							} else {
								caddyfileBlock.Merge(block)
							}
						}
					}
				}
			} else {
				logger.Error("Failed to get Swarm configs", zap.Error(err))
			}
		} else {
			logger.Debug("Skipping swarm config caddyfiles because swarm is not available")
		}

		// Add containers
		containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
		if err == nil {
			for _, container := range containers {
				if _, isControlledServer := container.Labels[g.options.ControlledServersLabel]; isControlledServer {
					ips, err := g.getContainerIPAddresses(&container, logger, g.controllerNetworks)
					if err != nil {
						logger.Error("Failed to get Container IPs", zap.String("container", container.ID), zap.Error(err))
					} else {
						controlledServers = append(controlledServers, ips...)
					}
				}

				containerCaddyfile, err := g.getContainerCaddyfile(&container, logger)
				if err == nil {
					caddyfileBlock.Merge(containerCaddyfile)
				} else {
					logger.Error("Failed to get Container Caddyfile", zap.String("container", container.ID), zap.Error(err))
				}
			}
		} else {
			logger.Error("Failed to get ContainerList", zap.Error(err))
		}

		// Add services
		if g.swarmIsAvailable[i] {
			services, err := dockerClient.ServiceList(context.Background(), types.ServiceListOptions{})
			if err == nil {
				for _, service := range services {
					logger.Debug("Swarm service", zap.String("service", service.Spec.Name))

					if _, isControlledServer := service.Spec.Labels[g.options.ControlledServersLabel]; isControlledServer {
						ips, err := g.getServiceTasksIps(&service, logger, g.controllerNetworks)
						if err != nil {
							logger.Error("Failed to  get Swarm service IPs", zap.String("service", service.Spec.Name), zap.Error(err))
						} else {
							controlledServers = append(controlledServers, ips...)
						}
					}

					serviceCaddyfile, err := g.getServiceCaddyfile(&service, logger)
					if err == nil {
						caddyfileBlock.Merge(serviceCaddyfile)
					} else {
						logger.Error("Failed to get Swarm service caddyfile", zap.String("service", service.Spec.Name), zap.Error(err))
					}
				}
			} else {
				logger.Error("Failed to get Swarm services", zap.Error(err))
			}
		} else {
			logger.Debug("Skipping swarm services because swarm is not available")
		}
	}

	// Write global blocks first
	globalCaddyfile := caddyfile.CreateContainer()
	for _, block := range caddyfileBlock.Children {
		if block.IsGlobalBlock() {
			globalCaddyfile.AddBlock(block)
			caddyfileBlock.Remove(block)
		}
	}
	caddyfileBuffer.Write(globalCaddyfile.Marshal())

	// Write remaining blocks
	caddyfileBuffer.Write(caddyfileBlock.Marshal())

	caddyfileContent := caddyfileBuffer.Bytes()

	if g.options.ProcessCaddyfile {
		processCaddyfileContent, processLogs := caddyfile.Process(caddyfileContent)
		caddyfileContent = processCaddyfileContent
		if len(processLogs) > 0 {
			logger.Info("Process Caddyfile", zap.ByteString("logs", processLogs))
		}
	}

	if len(caddyfileContent) == 0 {
		caddyfileContent = []byte("# Empty caddyfile")
	}

	if g.options.Mode&config.Server == config.Server {
		controlledServers = append(controlledServers, "localhost")
	}

	return caddyfileContent, controlledServers
}

// GetControllerNetworkGroup returns the controller network group
func (g *CaddyfileGenerator) GetControllerNetworkGroup(logger *zap.Logger) (*NetworkGroup, error) {
	return g.controllerNetworks, nil
}

func (g *CaddyfileGenerator) checkSwarmAvailability(logger *zap.Logger, isFirstCheck bool) {
	for i, dockerClient := range g.dockerClients {
		info, err := dockerClient.Info(context.Background())
		if err == nil {
			newSwarmIsAvailable := info.Swarm.LocalNodeState == swarm.LocalNodeStateActive
			if isFirstCheck || newSwarmIsAvailable != g.swarmIsAvailable[i] {
				logger.Info("Swarm is available", zap.Bool("new", newSwarmIsAvailable))
			}
			g.swarmIsAvailable[i] = newSwarmIsAvailable
		} else {
			logger.Error("Swarm availability check failed", zap.Error(err))
			g.swarmIsAvailable[i] = false
		}
	}
}

func (g *CaddyfileGenerator) createNetworkGroup(logger *zap.Logger, groupName string, input []string) (*NetworkGroup, error) {
	networkGroup := NetworkGroup{
		Name: groupName,
	}

	for _, dockerClient := range g.dockerClients {
		if len(g.options.IngressNetworks) > 0 {
			networks, err := dockerClient.NetworkList(context.Background(), types.NetworkListOptions{})
			if err != nil {
				return nil, err
			}
			for _, dockerNetwork := range networks {
				if dockerNetwork.Ingress {
					continue
				}
				foundNetwork := false
				for _, ingressNetwork := range g.options.IngressNetworks {
					if dockerNetwork.Name == ingressNetwork {
						foundNetwork = true
						networkInfo := NetworkInfo{
							ID:   dockerNetwork.ID,
							Name: dockerNetwork.Name,
						}
						for _, ipamConfig := range dockerNetwork.IPAM.Config {
							if _, ipNet, err := net.ParseCIDR(ipamConfig.Subnet); err == nil && ipNet != nil {
								networkInfo.Subnets = append(networkInfo.Subnets, *ipNet)
							}
						}
						networkGroup.Networks = append(networkGroup.Networks, &networkInfo)
					}
				}
				if !foundNetwork {
					if _, ipNet, err := net.ParseCIDR(g.options.ControllerNetwork); err == nil && ipNet != nil {
						networkInfo := NetworkInfo{
							Subnets: []net.IPNet{*ipNet},
						}
						networkGroup.Networks = append(networkGroup.Networks, &networkInfo)
					} else {
						logger.Warn("Controller network not found", zap.Any("network", g.ingressNetworks))
					}
				}
			}
		} else {
			containerID, err := g.dockerUtils.GetCurrentContainerID()
			if err != nil {
				return nil, err
			}
			logger.Info("Caddy ContainerID", zap.String("ID", containerID))
			container, err := dockerClient.ContainerInspect(context.Background(), containerID)
			if err != nil {
				return nil, err
			}

			for _, network := range container.NetworkSettings.Networks {
				dockerNetwork, err := dockerClient.NetworkInspect(context.Background(), network.NetworkID, types.NetworkInspectOptions{})
				if err != nil {
					return nil, err
				}
				if dockerNetwork.Ingress {
					continue
				}
				networkInfo := NetworkInfo{
					ID:   dockerNetwork.ID,
					Name: dockerNetwork.Name,
				}
				for _, ipamConfig := range dockerNetwork.IPAM.Config {
					if _, ipNet, err := net.ParseCIDR(ipamConfig.Subnet); err == nil && ipNet != nil {
						networkInfo.Subnets = append(networkInfo.Subnets, *ipNet)
					}
				}
				networkGroup.Networks = append(networkGroup.Networks, &networkInfo)
			}
		}
	}

	logger.Info("Network group created", zap.String("name", networkGroup.Name), zap.Any("networks", networkGroup.Networks))

	return &networkGroup, nil
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
