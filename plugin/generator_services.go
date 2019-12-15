package plugin

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
)

func (g *CaddyfileGenerator) getServiceDirectives(service *swarm.Service) (map[string]*directiveData, error) {
	return g.parseDirectives(service.Spec.Labels, service, func() ([]string, error) {
		return g.getServiceProxyTargets(service)
	})
}

func (g *CaddyfileGenerator) getServiceProxyTargets(service *swarm.Service) ([]string, error) {
	if g.proxyServiceTasks {
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
		if !g.validateNetwork || g.caddyNetworks[virtualIP.NetworkID] {
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
				if !g.validateNetwork || g.caddyNetworks[networkAttachment.Network.ID] {
					tasksIps = append(tasksIps, networkAttachment.Addresses...)
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
