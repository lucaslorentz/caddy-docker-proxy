package plugin

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// DockerClient is an interface with needed functionalities from docker client
type DockerClient interface {
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	ServiceList(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error)
	Info(ctx context.Context) (types.Info, error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	NetworkInspect(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, error)
}

func WrapDockerClient(client *client.Client) *dockerClientWrapper {
	return &dockerClientWrapper{
		client: client,
	}
}

type dockerClientWrapper struct {
	client *client.Client
}

func (wrapper *dockerClientWrapper) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	return wrapper.client.ContainerList(ctx, options)
}

func (wrapper *dockerClientWrapper) ServiceList(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {
	return wrapper.client.ServiceList(ctx, options)
}

func (wrapper *dockerClientWrapper) Info(ctx context.Context) (types.Info, error) {
	return wrapper.client.Info(ctx)
}

func (wrapper *dockerClientWrapper) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return wrapper.client.ContainerInspect(ctx, containerID)
}

func (wrapper *dockerClientWrapper) NetworkInspect(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, error) {
	return wrapper.client.NetworkInspect(ctx, networkID, options)
}
