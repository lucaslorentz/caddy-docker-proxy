package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lucaslorentz/caddy-docker-proxy/v2/caddyfile"
	"github.com/moby/moby/api/types/container"
	"go.uber.org/zap"
)

func (g *CaddyfileGenerator) getContainerCaddyfile(container *container.Summary, logger *zap.Logger) (*caddyfile.Container, error) {
	caddyLabels := g.filterLabels(container.Labels)

	return labelsToCaddyfile(
		caddyLabels,
		container,
		func() ([]string, error) {
			return g.getContainerIPAddresses(container, logger, true)
		},
		func() (int, error) {
			return getContainerLowestExposedPort(container)
		},
	)
}

// getContainerLowestExposedPort returns the lowest TCP port exposed by the container.
func getContainerLowestExposedPort(container *container.Summary) (int, error) {
	lowest := 0
	for _, port := range container.Ports {
		if port.Type == "tcp" && (lowest == 0 || int(port.PrivatePort) < lowest) {
			lowest = int(port.PrivatePort)
		}
	}
	if lowest == 0 {
		return 0, fmt.Errorf("container %s doesn't expose any TCP port", containerName(container))
	}
	return lowest, nil
}

func (g *CaddyfileGenerator) getContainerIPAddresses(container *container.Summary, logger *zap.Logger, onlyIngressIps bool) ([]string, error) {
	ips := []string{}
	inIngressNetwork := false

	ingressNetworkFromLabel, overrideNetwork := container.Labels[IngressNetworkLabel]

	for networkName, network := range container.NetworkSettings.Networks {
		include := false

		if !onlyIngressIps {
			include = true
		} else if overrideNetwork {
			include = networkName == ingressNetworkFromLabel
		} else {
			include = g.ingressNetworks[network.NetworkID] || g.ingressNetworks[networkName]
		}

		if include {
			inIngressNetwork = true
			if network.IPAddress.IsValid() {
				ips = append(ips, network.IPAddress.String())
			}
		}
	}

	// Warns when the container shares no ingress network with caddy.
	if !inIngressNetwork {
		networks := make([]string, 0, len(container.NetworkSettings.Networks))
		for networkName := range container.NetworkSettings.Networks {
			networks = append(networks, networkName)
		}
		sort.Strings(networks)
		logger.Warn("Container is not in same network as caddy",
			zap.String("container", containerName(container)),
			zap.Strings("container networks", networks),
			zap.Strings("ingress networks", g.options.IngressNetworks),
		)
	}

	return ips, nil
}

// containerName returns a human-friendly container name (without Docker's
// leading slash), falling back to the container ID.
func containerName(container *container.Summary) string {
	if len(container.Names) > 0 {
		return strings.TrimPrefix(container.Names[0], "/")
	}
	return container.ID
}
