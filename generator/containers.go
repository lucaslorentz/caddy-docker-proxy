package generator

import (
	"sort"
	"strings"

	"github.com/lucaslorentz/caddy-docker-proxy/v2/caddyfile"
	"github.com/moby/moby/api/types/container"
	"go.uber.org/zap"
)

func (g *CaddyfileGenerator) getContainerCaddyfile(container *container.Summary, logger *zap.Logger) (*caddyfile.Container, error) {
	caddyLabels := g.filterLabels(container.Labels)

	return labelsToCaddyfile(caddyLabels, container, func() ([]string, error) {
		return g.getContainerIPAddresses(container, logger, true)
	})
}

func (g *CaddyfileGenerator) getContainerIPAddresses(container *container.Summary, logger *zap.Logger, onlyIngressIps bool) ([]string, error) {
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

		if include && network.IPAddress.IsValid() {
			ips = append(ips, network.IPAddress.String())
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
func containerName(container *container.Summary) string {
	if len(container.Names) > 0 {
		return strings.TrimPrefix(container.Names[0], "/")
	}
	return container.ID
}
