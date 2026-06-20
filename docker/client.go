package docker

import (
	"context"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
)

// Client is an interface with needed functionalities from docker client
type Client interface {
	ContainerList(ctx context.Context, options client.ContainerListOptions) ([]container.Summary, error)
	ServiceList(ctx context.Context, options client.ServiceListOptions) ([]swarm.Service, error)
	TaskList(ctx context.Context, options client.TaskListOptions) ([]swarm.Task, error)
	Info(ctx context.Context) (system.Info, error)
	ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error)
	NetworkInspect(ctx context.Context, networkID string, options client.NetworkInspectOptions) (network.Inspect, error)
	NetworkList(ctx context.Context, options client.NetworkListOptions) ([]network.Summary, error)
	ConfigList(ctx context.Context, options client.ConfigListOptions) ([]swarm.Config, error)
	ConfigInspectWithRaw(ctx context.Context, id string) (swarm.Config, []byte, error)
	Events(ctx context.Context, options client.EventsListOptions) (<-chan events.Message, <-chan error)
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

func (wrapper *clientWrapper) ContainerList(ctx context.Context, options client.ContainerListOptions) ([]container.Summary, error) {
	result, err := wrapper.client.ContainerList(ctx, options)
	return result.Items, err
}

func (wrapper *clientWrapper) ServiceList(ctx context.Context, options client.ServiceListOptions) ([]swarm.Service, error) {
	result, err := wrapper.client.ServiceList(ctx, options)
	return result.Items, err
}

func (wrapper *clientWrapper) TaskList(ctx context.Context, options client.TaskListOptions) ([]swarm.Task, error) {
	result, err := wrapper.client.TaskList(ctx, options)
	return result.Items, err
}

func (wrapper *clientWrapper) ConfigList(ctx context.Context, options client.ConfigListOptions) ([]swarm.Config, error) {
	result, err := wrapper.client.ConfigList(ctx, options)
	return result.Items, err
}

func (wrapper *clientWrapper) Info(ctx context.Context) (system.Info, error) {
	result, err := wrapper.client.Info(ctx, client.InfoOptions{})
	return result.Info, err
}

func (wrapper *clientWrapper) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	result, err := wrapper.client.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	return result.Container, err
}

func (wrapper *clientWrapper) NetworkInspect(ctx context.Context, networkID string, options client.NetworkInspectOptions) (network.Inspect, error) {
	result, err := wrapper.client.NetworkInspect(ctx, networkID, options)
	return result.Network, err
}

func (wrapper *clientWrapper) NetworkList(ctx context.Context, options client.NetworkListOptions) ([]network.Summary, error) {
	result, err := wrapper.client.NetworkList(ctx, options)
	return result.Items, err
}

func (wrapper *clientWrapper) ConfigInspectWithRaw(ctx context.Context, id string) (swarm.Config, []byte, error) {
	result, err := wrapper.client.ConfigInspect(ctx, id, client.ConfigInspectOptions{})
	return result.Config, result.Raw, err
}

func (wrapper *clientWrapper) Events(ctx context.Context, options client.EventsListOptions) (<-chan events.Message, <-chan error) {
	result := wrapper.client.Events(ctx, options)
	return result.Messages, result.Err
}
