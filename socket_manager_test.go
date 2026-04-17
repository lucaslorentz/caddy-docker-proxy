package caddydockerproxy

import (
	"sync"
	"testing"
	"time"

	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/docker"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/utils"
	"github.com/stretchr/testify/assert"
)

// mockDockerClientFactory tracks connection attempts
type mockDockerClientFactory struct {
	mu        sync.Mutex
	attempts  int
	failUntil int
	client    docker.Client
}

func (f *mockDockerClientFactory) connect() (docker.Client, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attempts++
	if f.attempts <= f.failUntil {
		return nil, assert.AnError
	}
	return f.client, nil
}

func (f *mockDockerClientFactory) getAttempts() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.attempts
}

func TestSocketManagerConnectsSuccessfully(t *testing.T) {
	mockClient := &docker.ClientMock{}
	factory := &mockDockerClientFactory{client: mockClient, failUntil: 0}

	sm := &socketManager{
		socket:    "unix:///var/run/docker.sock",
		retryMin:  10 * time.Millisecond,
		retryMax:  100 * time.Millisecond,
		connectFn: factory.connect,
	}

	client, backoff := sm.attemptConnect()
	assert.NotNil(t, client)
	assert.Equal(t, sm.retryMin, backoff)
	assert.Equal(t, 1, factory.getAttempts())
}

func TestSocketManagerConnectReturnsNilOnFailure(t *testing.T) {
	factory := &mockDockerClientFactory{failUntil: 10}

	sm := &socketManager{
		socket:    "unix:///var/run/docker.sock",
		retryMin:  10 * time.Millisecond,
		retryMax:  100 * time.Millisecond,
		connectFn: factory.connect,
	}

	client, backoff := sm.attemptConnect()
	assert.Nil(t, client)
	assert.Equal(t, sm.retryMin, backoff)
	assert.Equal(t, 1, factory.getAttempts())
}

func TestSocketManagerBackoffDoublesOnFailure(t *testing.T) {
	sm := &socketManager{
		socket:   "unix:///var/run/docker.sock",
		retryMin: 10 * time.Millisecond,
		retryMax: 100 * time.Millisecond,
	}

	backoff := sm.retryMin
	backoff = sm.nextBackoff(backoff)
	assert.Equal(t, 20*time.Millisecond, backoff)

	backoff = sm.nextBackoff(backoff)
	assert.Equal(t, 40*time.Millisecond, backoff)

	backoff = sm.nextBackoff(backoff)
	assert.Equal(t, 80*time.Millisecond, backoff)

	// Should cap at retryMax
	backoff = sm.nextBackoff(backoff)
	assert.Equal(t, 100*time.Millisecond, backoff)

	// Should stay at retryMax
	backoff = sm.nextBackoff(backoff)
	assert.Equal(t, 100*time.Millisecond, backoff)
}

func TestDockerLoaderGetConnectedClients(t *testing.T) {
	loader := &DockerLoader{
		options:         &config.Options{},
		serversVersions: utils.NewStringInt64CMap(),
		serversUpdating: utils.NewStringBoolCMap(),
	}

	mockClient1 := &docker.ClientMock{}
	mockClient2 := &docker.ClientMock{}

	sm1 := &socketManager{client: mockClient1}
	sm2 := &socketManager{client: nil} // disconnected
	sm3 := &socketManager{client: mockClient2}

	loader.socketManagers = []*socketManager{sm1, sm2, sm3}

	clients := loader.getConnectedClients()
	assert.Len(t, clients, 2)
	assert.Equal(t, mockClient1, clients[0])
	assert.Equal(t, mockClient2, clients[1])
}

func TestDockerLoaderGetConnectedClientsEmpty(t *testing.T) {
	loader := &DockerLoader{
		options:         &config.Options{},
		serversVersions: utils.NewStringInt64CMap(),
		serversUpdating: utils.NewStringBoolCMap(),
	}
	loader.socketManagers = []*socketManager{}

	clients := loader.getConnectedClients()
	assert.Len(t, clients, 0)
}
