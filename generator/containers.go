package generator

import (
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
			include = g.ingressNetworks[network.NetworkID] || g.ingressNetworks[networkName]
		}

		if include {
			var ipAddress string
			if networkName == "host" && network.IPAddress == "" {
				ipAddress = "127.0.0.1"
			} else {
				ipAddress = network.IPAddress
			}
			ips = append(ips, ipAddress)
		}
	}

	if len(ips) == 0 {
		logger.Warn("Container is not in same network as caddy", zap.String("container", container.ID), zap.String("container id", container.ID))
	}

	return ips, nil
}
