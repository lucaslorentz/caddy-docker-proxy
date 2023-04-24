package generator

import (
	"context"
	"net"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/caddyfile"

	"go.uber.org/zap"
)

func (g *CaddyfileGenerator) getServiceCaddyfile(service *swarm.Service, logger *zap.Logger) (*caddyfile.Container, error) {
	caddyLabels := g.filterLabels(service.Spec.Labels)

	return labelsToCaddyfile(caddyLabels, service, func() ([]string, error) {
		return g.getServiceProxyTargets(service, logger, g.ingressNetworks)
	})
}

func (g *CaddyfileGenerator) getServiceProxyTargets(service *swarm.Service, logger *zap.Logger, networkGroup *NetworkGroup) ([]string, error) {
	if g.options.ProxyServiceTasks {
		return g.getServiceTasksIps(service, logger, networkGroup)
	}

	_, err := g.getServiceVirtualIps(service, logger, networkGroup)
	if err != nil {
		return nil, err
	}

	return []string{service.Spec.Name}, nil
}

func (g *CaddyfileGenerator) getServiceVirtualIps(service *swarm.Service, logger *zap.Logger, networkGroup *NetworkGroup) ([]string, error) {
	virtualIps := []string{}

	for _, virtualIP := range service.Endpoint.VirtualIPs {
		if networkGroup == nil ||
			networkGroup.MatchesID(virtualIP.NetworkID) ||
			networkGroup.MatchesName(virtualIP.NetworkID) ||
			networkGroup.ContainsIP(net.ParseIP(virtualIP.Addr)) {
			virtualIps = append(virtualIps, virtualIP.Addr)
		}
	}

	if len(virtualIps) == 0 {
		logger.Warn("Service is not in network group", zap.String("service", service.Spec.Name), zap.String("serviceId", service.ID), zap.Any("networkGroup", networkGroup))
	}

	return virtualIps, nil
}

func (g *CaddyfileGenerator) getServiceTasksIps(service *swarm.Service, logger *zap.Logger, networkGroup *NetworkGroup) ([]string, error) {
	taskListFilter := filters.NewArgs()
	taskListFilter.Add("service", service.ID)
	taskListFilter.Add("desired-state", "running")

	hasRunningTasks := false
	tasksIps := []string{}

	for _, dockerClient := range g.dockerClients {
		tasks, err := dockerClient.TaskList(context.Background(), types.TaskListOptions{Filters: taskListFilter})
		if err != nil {
			return []string{}, err
		}

		for _, task := range tasks {
			if task.Status.State == swarm.TaskStateRunning {
				hasRunningTasks = true
				for _, networkAttachment := range task.NetworksAttachments {
					for _, address := range networkAttachment.Addresses {
						ip, _, _ := net.ParseCIDR(address)
						if networkGroup == nil ||
							networkGroup.MatchesID(networkAttachment.Network.ID) ||
							networkGroup.ContainsIP(ip) {
							tasksIps = append(tasksIps, ip.String())
						}
					}
				}
			}
		}
	}

	if !hasRunningTasks {
		logger.Warn("Service has no tasks in running state", zap.String("service", service.Spec.Name), zap.String("serviceId", service.ID))

	} else if len(tasksIps) == 0 {
		logger.Warn("Service is not in network group", zap.String("service", service.Spec.Name), zap.String("serviceId", service.ID), zap.Any("networkGroup", networkGroup))
	}

	return tasksIps, nil
}
