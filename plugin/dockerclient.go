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
	TaskList(ctx context.Context, options types.TaskListOptions) ([]swarm.Task, error)
	Info(ctx context.Context) (types.Info, error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	NetworkInspect(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, error)
	ConfigList(ctx context.Context, options types.ConfigListOptions) ([]swarm.Config, error)
	ConfigInspectWithRaw(ctx context.Context, id string) (swarm.Config, []byte, error)
}

// WrapDockerClient creates a new docker client wrapper
func WrapDockerClient(client *client.Client) DockerClient {
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

func (wrapper *dockerClientWrapper) TaskList(ctx context.Context, options types.TaskListOptions) ([]swarm.Task, error) {
	return wrapper.client.TaskList(ctx, options)
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

func (wrapper *dockerClientWrapper) ConfigList(ctx context.Context, options types.ConfigListOptions) ([]swarm.Config, error) {
	return wrapper.client.ConfigList(ctx, options)
}

func (wrapper *dockerClientWrapper) ConfigInspectWithRaw(ctx context.Context, id string) (swarm.Config, []byte, error) {
	return wrapper.client.ConfigInspectWithRaw(ctx, id)
}
