package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// Client is an interface with needed functionalities from docker client
type Client interface {
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	ServiceList(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error)
	TaskList(ctx context.Context, options types.TaskListOptions) ([]swarm.Task, error)
	Info(ctx context.Context) (types.Info, error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	NetworkInspect(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, error)
	NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error)
	ConfigList(ctx context.Context, options types.ConfigListOptions) ([]swarm.Config, error)
	ConfigInspectWithRaw(ctx context.Context, id string) (swarm.Config, []byte, error)
	Events(ctx context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error)
}

// WrapClient creates a new docker client wrapper
func WrapClient(client *client.Client) Client {
	return &clientWrapper{
		client: client,
	}
}

type clientWrapper struct {
	client *client.Client
}

func (wrapper *clientWrapper) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	return wrapper.client.ContainerList(ctx, options)
}

func (wrapper *clientWrapper) ServiceList(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {
	return wrapper.client.ServiceList(ctx, options)
}

func (wrapper *clientWrapper) TaskList(ctx context.Context, options types.TaskListOptions) ([]swarm.Task, error) {
	return wrapper.client.TaskList(ctx, options)
}

func (wrapper *clientWrapper) ConfigList(ctx context.Context, options types.ConfigListOptions) ([]swarm.Config, error) {
	return wrapper.client.ConfigList(ctx, options)
}

func (wrapper *clientWrapper) Info(ctx context.Context) (types.Info, error) {
	return wrapper.client.Info(ctx)
}

func (wrapper *clientWrapper) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return wrapper.client.ContainerInspect(ctx, containerID)
}

func (wrapper *clientWrapper) NetworkInspect(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, error) {
	return wrapper.client.NetworkInspect(ctx, networkID, options)
}

func (wrapper *clientWrapper) NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
	return wrapper.client.NetworkList(ctx, options)
}

func (wrapper *clientWrapper) ConfigInspectWithRaw(ctx context.Context, id string) (swarm.Config, []byte, error) {
	return wrapper.client.ConfigInspectWithRaw(ctx, id)
}

func (wrapper *clientWrapper) Events(ctx context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error) {
	return wrapper.client.Events(ctx, options)
}
