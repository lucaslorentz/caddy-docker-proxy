package generator

import (
	"bytes"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/lucaslorentz/caddy-docker-proxy/plugin/v2/caddyfile"
)

func (g *CaddyfileGenerator) getContainerCaddyfile(container *types.Container, logsBuffer *bytes.Buffer) (*caddyfile.Container, error) {
	caddyLabels := g.filterLabels(container.Labels)

	return labelsToCaddyfile(caddyLabels, container, func() ([]string, error) {
		return g.getContainerIPAddresses(container, logsBuffer, true)
	})
}

func (g *CaddyfileGenerator) getContainerIPAddresses(container *types.Container, logsBuffer *bytes.Buffer, ingress bool) ([]string, error) {
	ips := []string{}

	for _, network := range container.NetworkSettings.Networks {
		if !ingress || !g.options.ValidateNetwork || g.ingressNetworks[network.NetworkID] {
			ips = append(ips, network.IPAddress)
		}
	}

	if len(ips) == 0 {
		logsBuffer.WriteString(fmt.Sprintf("[WARNING] Container %v and caddy are not in same network\n", container.ID))
	}

	return ips, nil
}
