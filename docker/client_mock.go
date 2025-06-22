package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
)

// ClientMock allows easily mocking of docker client data
type ClientMock struct {
	ContainersData       []types.Container
	ServicesData         []swarm.Service
	ConfigsData          []swarm.Config
	TasksData            []swarm.Task
	NetworksData         []network.Summary
	InfoData             system.Info
	ContainerInspectData map[string]types.ContainerJSON
	NetworkInspectData   map[string]network.Inspect
	EventsChannel        chan events.Message
	ErrorsChannel        chan error
}

// ContainerList list all containers
func (mock *ClientMock) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return mock.ContainersData, nil
}

// ServiceList list all services
func (mock *ClientMock) ServiceList(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {
	return mock.ServicesData, nil
}

// TaskList list all tasks
func (mock *ClientMock) TaskList(ctx context.Context, options types.TaskListOptions) ([]swarm.Task, error) {
	matchingTasks := []swarm.Task{}
	for _, task := range mock.TasksData {
		if !options.Filters.Match("service", task.ServiceID) {
			continue
		}
		if !options.Filters.Match("desired-state", string(task.DesiredState)) {
			continue
		}
		matchingTasks = append(matchingTasks, task)
	}
	return matchingTasks, nil
}

// ConfigList list all configs
func (mock *ClientMock) ConfigList(ctx context.Context, options types.ConfigListOptions) ([]swarm.Config, error) {
	return mock.ConfigsData, nil
}

// NetworkList list all networks
func (mock *ClientMock) NetworkList(ctx context.Context, options network.ListOptions) ([]network.Summary, error) {
	return mock.NetworksData, nil
}

// Info retrieves information about docker host
func (mock *ClientMock) Info(ctx context.Context) (system.Info, error) {
	return mock.InfoData, nil
}

// ContainerInspect returns information about a specific container
func (mock *ClientMock) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return mock.ContainerInspectData[containerID], nil
}

// NetworkInspect returns information about a specific network
func (mock *ClientMock) NetworkInspect(ctx context.Context, networkID string, options network.InspectOptions) (network.Inspect, error) {
	return mock.NetworkInspectData[networkID], nil
}

// ConfigInspectWithRaw return sinformation about a specific config
func (mock *ClientMock) ConfigInspectWithRaw(ctx context.Context, id string) (swarm.Config, []byte, error) {
	for _, config := range mock.ConfigsData {
		if config.ID == id {
			return config, nil, nil
		}
	}
	return swarm.Config{}, nil, nil
}

// Events listen for events in docker
func (mock *ClientMock) Events(ctx context.Context, options events.ListOptions) (<-chan events.Message, <-chan error) {
	return mock.EventsChannel, mock.ErrorsChannel
}
