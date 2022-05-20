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
		return g.getServiceProxyTargets(service, logger, true)
	})
}

func (g *CaddyfileGenerator) getServiceProxyTargets(service *swarm.Service, logger *zap.Logger, ingress bool) ([]string, error) {
	if g.options.ProxyServiceTasks {
		return g.getServiceTasksIps(service, logger, ingress)
	}

	_, err := g.getServiceVirtualIps(service, logger, ingress)
	if err != nil {
		return nil, err
	}

	return []string{service.Spec.Name}, nil
}

func (g *CaddyfileGenerator) getServiceVirtualIps(service *swarm.Service, logger *zap.Logger, ingress bool) ([]string, error) {
	virtualIps := []string{}

	for _, virtualIP := range service.Endpoint.VirtualIPs {
		if !ingress || g.ingressNetworks[virtualIP.NetworkID] {
			virtualIps = append(virtualIps, virtualIP.Addr)
		}
	}

	if len(virtualIps) == 0 {
		logger.Warn("Service is not in same network as caddy", zap.String("service", service.Spec.Name), zap.String("serviceId", service.ID))
	}

	return virtualIps, nil
}

func (g *CaddyfileGenerator) getServiceTasksIps(service *swarm.Service, logger *zap.Logger, ingress bool) ([]string, error) {
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
					if !ingress || g.ingressNetworks[networkAttachment.Network.ID] {
						for _, address := range networkAttachment.Addresses {
							ipAddress, _, _ := net.ParseCIDR(address)
							tasksIps = append(tasksIps, ipAddress.String())
						}
					}
				}
			}
		}
	}

	if !hasRunningTasks {
		logger.Warn("Service has no tasks in running state", zap.String("service", service.Spec.Name), zap.String("serviceId", service.ID))

	} else if len(tasksIps) == 0 {
		logger.Warn("Service is not in same network as caddy", zap.String("service", service.Spec.Name), zap.String("serviceId", service.ID))
	}

	return tasksIps, nil
}
