package generator

import (
	"net"

	"github.com/docker/docker/api/types"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/caddyfile"
	"go.uber.org/zap"
)

func (g *CaddyfileGenerator) getContainerCaddyfile(container *types.Container, logger *zap.Logger) (*caddyfile.Container, error) {
	caddyLabels := g.filterLabels(container.Labels)

	return labelsToCaddyfile(caddyLabels, container, func() ([]string, error) {
		return g.getContainerIPAddresses(container, logger, g.ingressNetworks)
	})
}

func (g *CaddyfileGenerator) getContainerIPAddresses(container *types.Container, logger *zap.Logger, networkGroup *NetworkGroup) ([]string, error) {
	ips := []string{}

	for _, network := range container.NetworkSettings.Networks {
		if networkGroup == nil ||
			networkGroup.MatchesID(network.NetworkID) ||
			networkGroup.MatchesName(network.NetworkID) ||
			networkGroup.ContainsIP(net.ParseIP(network.IPAddress)) {
			ips = append(ips, network.IPAddress)
		}
	}

	if len(ips) == 0 {
		logger.Warn("Container is not in network group", zap.Strings("container", container.Names), zap.String("containerId", container.ID), zap.Any("networkGroup", networkGroup))

	}

	return ips, nil
}
