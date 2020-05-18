package generator

import (
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/plugin/caddyfile"
)

func (g *CaddyfileGenerator) getContainerCaddyfile(container *types.Container) (*caddyfile.Block, error) {
	caddyLabels := g.filterLabels(container.Labels)

	return labelsToCaddyfile(caddyLabels, container, func() ([]string, error) {
		return g.getContainerIPAddresses(container)
	})
}

func (g *CaddyfileGenerator) getContainerIPAddresses(container *types.Container) ([]string, error) {
	ips := []string{}

	for _, network := range container.NetworkSettings.Networks {
		if !g.options.ValidateNetwork || g.caddyNetworks[network.NetworkID] {
			ips = append(ips, network.IPAddress)
		}
	}

	if len(ips) == 0 {
		return ips, fmt.Errorf("Container %v and caddy are not in same network", container.ID)
	}

	return ips, nil
}
