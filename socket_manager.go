package caddydockerproxy

import (
	"context"
	"os"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/docker"

	"go.uber.org/zap"
)

// connectFn is a function that creates and pings a docker client.
// Returns the connected client or an error.
type connectFn func() (docker.Client, error)

type socketManager struct {
	socket     string
	certsPath  string
	apiVersion string
	retryMin   time.Duration
	retryMax   time.Duration

	connectFn  connectFn
	client     docker.Client
	skipEvents bool

	loader *DockerLoader
}

// newSocketManager creates a socket manager for the given socket address.
func newSocketManager(socket, certsPath, apiVersion string, retryMin, retryMax time.Duration, loader *DockerLoader) *socketManager {
	sm := &socketManager{
		socket:     socket,
		certsPath:  certsPath,
		apiVersion: apiVersion,
		retryMin:   retryMin,
		retryMax:   retryMax,
		loader:     loader,
	}
	sm.connectFn = sm.defaultConnect
	return sm
}

// defaultConnect creates a docker client using explicit options (no env vars).
func (sm *socketManager) defaultConnect() (docker.Client, error) {
	opts := []client.Opt{
		client.WithHost(sm.socket),
		client.WithAPIVersionNegotiation(),
	}

	if sm.certsPath != "" {
		opts = append(opts, client.WithTLSClientConfig(
			sm.certsPath+"/ca.pem",
			sm.certsPath+"/cert.pem",
			sm.certsPath+"/key.pem",
		))
	}

	if sm.apiVersion != "" {
		opts = append(opts, client.WithVersion(sm.apiVersion))
	}

	dockerClient, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}

	_, err = dockerClient.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	return docker.WrapClient(dockerClient), nil
}

// attemptConnect tries to connect once. Returns the client (or nil) and the
// backoff duration to use before the next attempt.
func (sm *socketManager) attemptConnect() (docker.Client, time.Duration) {
	log := logger()

	c, err := sm.connectFn()
	if err != nil {
		log.Warn("Docker socket unavailable, will retry",
			zap.String("socket", sm.socket),
			zap.Error(err),
		)
		return nil, sm.retryMin
	}

	log.Info("Docker socket connected", zap.String("socket", sm.socket))
	return c, sm.retryMin
}

// nextBackoff doubles the current backoff, capped at retryMax.
func (sm *socketManager) nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > sm.retryMax {
		next = sm.retryMax
	}
	return next
}

// run is the main lifecycle goroutine for this socket.
// It loops forever: connect -> listen for events -> on error, disconnect and retry.
func (sm *socketManager) run() {
	log := logger()
	backoff := sm.retryMin

	for {
		// Connect phase
		c, _ := sm.attemptConnect()
		if c == nil {
			time.Sleep(backoff)
			backoff = sm.nextBackoff(backoff)
			continue
		}

		// Connected — reset backoff and register
		backoff = sm.retryMin
		sm.loader.setClient(sm, c)
		sm.loader.triggerUpdate()

		// Event listening phase
		sm.listenEvents(c)

		// Disconnected — remove client and trigger regeneration
		log.Warn("Docker socket disconnected", zap.String("socket", sm.socket))
		sm.loader.setClient(sm, nil)
		sm.loader.triggerUpdate()
	}
}

// monitorEvents is used for sockets that are already connected at startup
// (the default single-socket path). It just listens for events and reconnects
// on error.
func (sm *socketManager) monitorEvents() {
	for {
		sm.listenEvents(sm.client)
		time.Sleep(30 * time.Second)
	}
}

// listenEvents listens for docker events on this socket's client.
// Returns when the event stream errors (indicating disconnection).
func (sm *socketManager) listenEvents(dockerClient docker.Client) {
	args := filters.NewArgs()
	if !isTrue.MatchString(os.Getenv("CADDY_DOCKER_NO_SCOPE")) {
		args.Add("scope", "swarm")
		args.Add("scope", "local")
	}
	args.Add("type", "service")
	args.Add("type", "container")
	args.Add("type", "config")
	args.Add("type", "network")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventsChan, errorChan := dockerClient.Events(ctx, events.ListOptions{
		Filters: args,
	})

	log := logger()
	log.Info("Listening for docker events", zap.String("socket", sm.socket))

	for {
		select {
		case event := <-eventsChan:
			if sm.skipEvents {
				continue
			}

			update := (event.Type == "container" && event.Action == "create") ||
				(event.Type == "container" && event.Action == "start") ||
				(event.Type == "container" && event.Action == "stop") ||
				(event.Type == "container" && event.Action == "die") ||
				(event.Type == "container" && event.Action == "destroy") ||
				(event.Type == "service" && event.Action == "create") ||
				(event.Type == "service" && event.Action == "update") ||
				(event.Type == "service" && event.Action == "remove") ||
				(event.Type == "config" && event.Action == "create") ||
				(event.Type == "config" && event.Action == "remove") ||
				(event.Type == "network" && event.Action == "connect") ||
				(event.Type == "network" && event.Action == "disconnect")

			if update {
				sm.skipEvents = true
				sm.loader.triggerThrottledUpdate()
			}
		case err := <-errorChan:
			if err != nil {
				log.Error("Docker events error", zap.String("socket", sm.socket), zap.Error(err))
			}
			return
		}
	}
}
