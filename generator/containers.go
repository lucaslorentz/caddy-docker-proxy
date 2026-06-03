package generator

import (
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/caddyfile"
	"go.uber.org/zap"
)

func (g *CaddyfileGenerator) getContainerCaddyfile(container *types.Container, logger *zap.Logger) (*caddyfile.Container, error) {
	caddyLabels := g.filterLabels(container.Labels)

	return labelsToCaddyfile(caddyLabels, container, func() ([]string, error) {
		return g.getContainerIPAddresses(container, logger, true)
	})
}

func (g *CaddyfileGenerator) getContainerIPAddresses(container *types.Container, logger *zap.Logger, onlyIngressIps bool) ([]string, error) {
	ips := []string{}

	ingressNetworkFromLabel, overrideNetwork := container.Labels[IngressNetworkLabel]

	for networkName, network := range container.NetworkSettings.Networks {
		include := false

		if !onlyIngressIps {
			include = true
		} else if overrideNetwork {
			include = networkName == ingressNetworkFromLabel
		} else {
			include = g.ingressNetworks[network.NetworkID]
		}

		if include {
			ips = append(ips, network.IPAddress)
		}
	}

	if len(ips) == 0 {
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
func containerName(container *types.Container) string {
	if len(container.Names) > 0 {
		return strings.TrimPrefix(container.Names[0], "/")
	}
	return container.ID
}
