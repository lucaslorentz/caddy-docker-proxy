package generator

import (
	"context"

	"github.com/lucaslorentz/caddy-docker-proxy/v2/caddyfile"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"

	"go.uber.org/zap"
)

func (g *CaddyfileGenerator) getServiceCaddyfile(service *swarm.Service, logger *zap.Logger) (*caddyfile.Container, error) {
	caddyLabels := g.filterLabels(service.Spec.Labels)

	return labelsToCaddyfile(caddyLabels, service, func() ([]string, error) {
		return g.getServiceProxyTargets(service, logger, true)
	})
}

func (g *CaddyfileGenerator) getServiceProxyTargets(service *swarm.Service, logger *zap.Logger, onlyIngressIps bool) ([]string, error) {
	if g.options.ProxyServiceTasks {
		return g.getServiceTasksIps(service, logger, onlyIngressIps)
	}

	_, err := g.getServiceVirtualIps(service, logger, onlyIngressIps)
	if err != nil {
		return nil, err
	}

	return []string{service.Spec.Name}, nil
}

func (g *CaddyfileGenerator) getServiceVirtualIps(service *swarm.Service, logger *zap.Logger, onlyIngressIps bool) ([]string, error) {
	virtualIps := []string{}

	for _, virtualIP := range service.Endpoint.VirtualIPs {
		if !onlyIngressIps || g.ingressNetworks[virtualIP.NetworkID] {
			virtualIps = append(virtualIps, virtualIP.Addr.String())
		}
	}

	if len(virtualIps) == 0 {
		logger.Warn("Service is not in same network as caddy", zap.String("service", service.Spec.Name), zap.String("serviceId", service.ID))
	}

	return virtualIps, nil
}

func (g *CaddyfileGenerator) getServiceTasksIps(service *swarm.Service, logger *zap.Logger, onlyIngressIps bool) ([]string, error) {
	taskListFilter := make(client.Filters)
	taskListFilter.Add("service", service.ID)
	taskListFilter.Add("desired-state", "running")

	hasRunningTasks := false
	tasksIps := []string{}

	for _, dockerClient := range g.dockerClients {
		tasks, err := dockerClient.TaskList(context.Background(), client.TaskListOptions{Filters: taskListFilter})
		if err != nil {
			logger.Debug("Failed to get Swarm tasks from docker client, skipping", zap.String("service", service.Spec.Name), zap.Error(err))
			continue
		}

		for _, task := range tasks {
			if task.Status.State == swarm.TaskStateRunning {
				hasRunningTasks = true
				ingressNetworkFromLabel, overrideNetwork := service.Spec.Labels[IngressNetworkLabel]

				for _, networkAttachment := range task.NetworksAttachments {
					include := false

					if !onlyIngressIps {
						include = true
					} else if overrideNetwork {
						include = networkAttachment.Network.Spec.Name == ingressNetworkFromLabel
					} else {
						include = g.ingressNetworks[networkAttachment.Network.ID]
					}

					if include {
						for _, address := range networkAttachment.Addresses {
							tasksIps = append(tasksIps, address.Addr().String())
						}
					}
				}
			}
		}
	}

	if !hasRunningTasks {
		logger.Debug("Service has no tasks in running state", zap.String("service", service.Spec.Name), zap.String("serviceId", service.ID))

	} else if len(tasksIps) == 0 {
		logger.Warn("Service is not in same network as caddy", zap.String("service", service.Spec.Name), zap.String("serviceId", service.ID))
	}

	return tasksIps, nil
}
