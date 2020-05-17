package generator

import (
	"context"
	"fmt"
	"net"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/plugin/caddyfile"
)

func (g *CaddyfileGenerator) getServiceCaddyfile(service *swarm.Service) (*caddyfile.Block, error) {
	caddyLabels := g.filterLabels(service.Spec.Labels)

	return labelsToCaddyfile(caddyLabels, service, func() ([]string, error) {
		return g.getServiceProxyTargets(service)
	})
}

func (g *CaddyfileGenerator) getServiceProxyTargets(service *swarm.Service) ([]string, error) {
	if g.options.ProxyServiceTasks {
		return g.getServiceTasksIps(service)
	}

	_, err := g.getServiceVirtualIps(service)
	if err != nil {
		return nil, err
	}

	return []string{service.Spec.Name}, nil
}

func (g *CaddyfileGenerator) getServiceVirtualIps(service *swarm.Service) ([]string, error) {
	virtualIps := []string{}

	for _, virtualIP := range service.Endpoint.VirtualIPs {
		if !g.options.ValidateNetwork || g.caddyNetworks[virtualIP.NetworkID] {
			virtualIps = append(virtualIps, virtualIP.Addr)
		}
	}

	if len(virtualIps) == 0 {
		return []string{}, fmt.Errorf("Service %v and caddy are not in same network", service.ID)
	}

	return virtualIps, nil
}

func (g *CaddyfileGenerator) getServiceTasksIps(service *swarm.Service) ([]string, error) {
	taskListFilter := filters.NewArgs()
	taskListFilter.Add("service", service.ID)
	taskListFilter.Add("desired-state", "running")

	tasks, err := g.dockerClient.TaskList(context.Background(), types.TaskListOptions{Filters: taskListFilter})
	if err != nil {
		return []string{}, err
	}

	hasRunningTasks := false
	tasksIps := []string{}
	for _, task := range tasks {
		if task.Status.State == swarm.TaskStateRunning {
			hasRunningTasks = true
			for _, networkAttachment := range task.NetworksAttachments {
				if !g.options.ValidateNetwork || g.caddyNetworks[networkAttachment.Network.ID] {
					for _, address := range networkAttachment.Addresses {
						ipAddress, _, _ := net.ParseCIDR(address)
						tasksIps = append(tasksIps, ipAddress.String())
					}
				}
			}
		}
	}

	if !hasRunningTasks {
		return []string{}, fmt.Errorf("Service %v doesn't have any task in running state", service.ID)
	}

	if len(tasksIps) == 0 {
		return []string{}, fmt.Errorf("Service %v and caddy are not in same network", service.ID)
	}

	return tasksIps, nil
}
