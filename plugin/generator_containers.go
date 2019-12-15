package plugin

import (
	"fmt"

	"github.com/docker/docker/api/types"
)

func (g *CaddyfileGenerator) getContainerDirectives(container *types.Container) (map[string]*directiveData, error) {
	return g.parseDirectives(container.Labels, container, func() ([]string, error) {
		return g.getContainerIPAddresses(container)
	})
}

func (g *CaddyfileGenerator) getContainerIPAddresses(container *types.Container) ([]string, error) {
	ips := []string{}

	for _, network := range container.NetworkSettings.Networks {
		if !g.validateNetwork || g.caddyNetworks[network.NetworkID] {
			ips = append(ips, network.IPAddress)
		}
	}

	if len(ips) == 0 {
		return ips, fmt.Errorf("Container %v and caddy are not in same network", container.ID)
	}

	return ips, nil
}
