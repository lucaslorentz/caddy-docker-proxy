package generator

import (
	"github.com/docker/docker/api/types"
	"github.com/lucaslorentz/caddy-docker-proxy/plugin/caddyfile"
	"go.uber.org/zap"
)

func (g *CaddyfileGenerator) getContainerCaddyfile(container *types.Container, logger *zap.Logger) (*caddyfile.Container, error) {
	caddyLabels := g.filterLabels(container.Labels)

	return labelsToCaddyfile(caddyLabels, container, func() ([]string, error) {
		return g.getContainerIPAddresses(container, logger, true)
	})
}

func (g *CaddyfileGenerator) getContainerIPAddresses(container *types.Container, logger *zap.Logger, ingress bool) ([]string, error) {
	ips := []string{}

	for _, network := range container.NetworkSettings.Networks {
		if !ingress || g.ingressNetworks[network.NetworkID] {
			ips = append(ips, network.IPAddress)
		}
	}

	if len(ips) == 0 {
		logger.Warn("Container is not in same network as caddy", zap.String("container", container.ID), zap.String("container id", container.ID))

	}

	return ips, nil
}
