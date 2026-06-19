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

// ClientMock allows easily mocking of docker client data
type ClientMock struct {
	ContainersData       []container.Summary
	ServicesData         []swarm.Service
	ConfigsData          []swarm.Config
	TasksData            []swarm.Task
	TaskListErr          error
	NetworksData         []network.Summary
	InfoData             system.Info
	ContainerInspectData map[string]container.InspectResponse
	NetworkInspectData   map[string]network.Inspect
	EventsChannel        chan events.Message
	ErrorsChannel        chan error
}

// ContainerList list all containers
func (mock *ClientMock) ContainerList(ctx context.Context, options client.ContainerListOptions) ([]container.Summary, error) {
	return mock.ContainersData, nil
}

// ServiceList list all services
func (mock *ClientMock) ServiceList(ctx context.Context, options client.ServiceListOptions) ([]swarm.Service, error) {
	return mock.ServicesData, nil
}

// TaskList list all tasks
func (mock *ClientMock) TaskList(ctx context.Context, options client.TaskListOptions) ([]swarm.Task, error) {
	if mock.TaskListErr != nil {
		return nil, mock.TaskListErr
	}
	matchingTasks := []swarm.Task{}
	for _, task := range mock.TasksData {
		if !filterMatches(options.Filters, "service", task.ServiceID) {
			continue
		}
		if !filterMatches(options.Filters, "desired-state", string(task.DesiredState)) {
			continue
		}
		matchingTasks = append(matchingTasks, task)
	}
	return matchingTasks, nil
}

// ConfigList list all configs
func (mock *ClientMock) ConfigList(ctx context.Context, options client.ConfigListOptions) ([]swarm.Config, error) {
	return mock.ConfigsData, nil
}

// NetworkList list all networks
func (mock *ClientMock) NetworkList(ctx context.Context, options client.NetworkListOptions) ([]network.Summary, error) {
	return mock.NetworksData, nil
}

// Info retrieves information about docker host
func (mock *ClientMock) Info(ctx context.Context) (system.Info, error) {
	return mock.InfoData, nil
}

// ContainerInspect returns information about a specific container
func (mock *ClientMock) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return mock.ContainerInspectData[containerID], nil
}

// NetworkInspect returns information about a specific network
func (mock *ClientMock) NetworkInspect(ctx context.Context, networkID string, options client.NetworkInspectOptions) (network.Inspect, error) {
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
func (mock *ClientMock) Events(ctx context.Context, options client.EventsListOptions) (<-chan events.Message, <-chan error) {
	return mock.EventsChannel, mock.ErrorsChannel
}

func filterMatches(filters client.Filters, term string, value string) bool {
	values, ok := filters[term]
	if !ok {
		return true
	}
	return values[value]
}
