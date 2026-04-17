package caddydockerproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"os"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/docker/docker/client"
	"github.com/joho/godotenv"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/docker"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/generator"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/utils"

	"go.uber.org/zap"
)

var CaddyfileAutosavePath = filepath.Join(caddy.AppConfigDir(), "Caddyfile.autosave")

// DockerLoader generates caddy files from docker swarm information
type DockerLoader struct {
	options         *config.Options
	initialized     bool
	mu              sync.RWMutex
	socketManagers  []*socketManager
	generator       *generator.CaddyfileGenerator
	timer           *time.Timer
	lastCaddyfile   []byte
	lastJSONConfig  []byte
	lastVersion     int64
	serversVersions *utils.StringInt64CMap
	serversUpdating *utils.StringBoolCMap
}

// CreateDockerLoader creates a docker loader
func CreateDockerLoader(options *config.Options) *DockerLoader {
	return &DockerLoader{
		options:         options,
		serversVersions: utils.NewStringInt64CMap(),
		serversUpdating: utils.NewStringBoolCMap(),
	}
}

// getConnectedClients returns a snapshot of all currently connected docker clients.
func (dockerLoader *DockerLoader) getConnectedClients() []docker.Client {
	dockerLoader.mu.RLock()
	defer dockerLoader.mu.RUnlock()

	clients := []docker.Client{}
	for _, sm := range dockerLoader.socketManagers {
		if sm.client != nil {
			clients = append(clients, sm.client)
		}
	}
	return clients
}

// setClient sets or clears a socket manager's client under write lock.
func (dockerLoader *DockerLoader) setClient(sm *socketManager, c docker.Client) {
	dockerLoader.mu.Lock()
	defer dockerLoader.mu.Unlock()
	sm.client = c
}

// triggerUpdate triggers an immediate Caddyfile regeneration.
func (dockerLoader *DockerLoader) triggerUpdate() {
	dockerLoader.timer.Reset(0)
}

// triggerThrottledUpdate triggers a throttled Caddyfile regeneration.
func (dockerLoader *DockerLoader) triggerThrottledUpdate() {
	dockerLoader.timer.Reset(dockerLoader.options.EventThrottleInterval)
}

func logger() *zap.Logger {
	return caddy.Log().
		Named("docker-proxy")
}

// Start docker loader
func (dockerLoader *DockerLoader) Start() error {
	if dockerLoader.initialized {
		return nil
	}

	dockerLoader.initialized = true
	log := logger()

	if envFile := dockerLoader.options.EnvFile; envFile != "" {
		if err := godotenv.Load(dockerLoader.options.EnvFile); err != nil {
			log.Error("Load variables from environment file failed", zap.Error(err), zap.String("envFile", dockerLoader.options.EnvFile))
			return err
		}
		log.Info("environment file loaded", zap.String("envFile", dockerLoader.options.EnvFile))
	}

	if len(dockerLoader.options.DockerSockets) > 0 {
		// Multi-socket mode: each socket is optional with retry
		for i, dockerSocket := range dockerLoader.options.DockerSockets {
			certsPath := ""
			if len(dockerLoader.options.DockerCertsPath) >= i+1 {
				certsPath = dockerLoader.options.DockerCertsPath[i]
			}

			apiVersion := ""
			if len(dockerLoader.options.DockerAPIsVersion) >= i+1 {
				apiVersion = dockerLoader.options.DockerAPIsVersion[i]
			}

			sm := newSocketManager(
				dockerSocket,
				certsPath,
				apiVersion,
				dockerLoader.options.DockerRetryMin,
				dockerLoader.options.DockerRetryMax,
				dockerLoader,
			)
			dockerLoader.socketManagers = append(dockerLoader.socketManagers, sm)
		}
	} else {
		// Default single-socket mode: strict, fail on error (backwards compat)
		dockerClient, err := client.NewEnvClient()
		if err != nil {
			log.Error("Docker connection failed", zap.Error(err))
			return err
		}

		dockerPing, err := dockerClient.Ping(context.Background())
		if err != nil {
			log.Error("Docker ping failed", zap.Error(err))
			return err
		}

		dockerClient.NegotiateAPIVersionPing(dockerPing)
		wrappedClient := docker.WrapClient(dockerClient)

		// Create a socket manager that is already connected (no retry loop)
		sm := &socketManager{
			socket:   os.Getenv("DOCKER_HOST"),
			retryMin: dockerLoader.options.DockerRetryMin,
			retryMax: dockerLoader.options.DockerRetryMax,
			client:   wrappedClient,
			loader:   dockerLoader,
		}
		dockerLoader.socketManagers = append(dockerLoader.socketManagers, sm)
		dockerLoader.options.DockerSockets = append(dockerLoader.options.DockerSockets, sm.socket)
	}

	dockerLoader.generator = generator.CreateGenerator(
		dockerLoader.getConnectedClients,
		docker.CreateUtils(),
		dockerLoader.options,
	)

	log.Info(
		"Start",
		zap.String("CaddyfilePath", dockerLoader.options.CaddyfilePath),
		zap.String("EnvFile", dockerLoader.options.EnvFile),
		zap.String("LabelPrefix", dockerLoader.options.LabelPrefix),
		zap.Duration("PollingInterval", dockerLoader.options.PollingInterval),
		zap.Bool("ProxyServiceTasks", dockerLoader.options.ProxyServiceTasks),
		zap.Bool("ProcessCaddyfile", dockerLoader.options.ProcessCaddyfile),
		zap.Bool("ScanStoppedContainers", dockerLoader.options.ScanStoppedContainers),
		zap.String("IngressNetworks", fmt.Sprintf("%v", dockerLoader.options.IngressNetworks)),
		zap.Strings("DockerSockets", dockerLoader.options.DockerSockets),
		zap.Strings("DockerCertsPath", dockerLoader.options.DockerCertsPath),
		zap.Strings("DockerAPIsVersion", dockerLoader.options.DockerAPIsVersion),
		zap.Duration("DockerRetryMin", dockerLoader.options.DockerRetryMin),
		zap.Duration("DockerRetryMax", dockerLoader.options.DockerRetryMax),
		zap.String("CaddyfileAutosavePath", CaddyfileAutosavePath),
	)

	ready := make(chan struct{})
	dockerLoader.timer = time.AfterFunc(0, func() {
		<-ready
		dockerLoader.update()
	})
	close(ready)

	// Start socket manager goroutines for event monitoring
	for _, sm := range dockerLoader.socketManagers {
		if sm.client != nil {
			// Already connected (default single-socket path) — just monitor events
			go sm.monitorEvents()
		} else {
			// Not yet connected (multi-socket path) — run full lifecycle
			go sm.run()
		}
	}

	return nil
}

func (dockerLoader *DockerLoader) update() bool {
	dockerLoader.timer.Reset(dockerLoader.options.PollingInterval)
	for _, sm := range dockerLoader.socketManagers {
		sm.skipEvents = false
	}

	// Don't cache the logger more globally, it can change based on config reloads
	log := logger()
	caddyfile, controlledServers := dockerLoader.generator.GenerateCaddyfile(log)

	caddyfileChanged := !bytes.Equal(dockerLoader.lastCaddyfile, caddyfile)

	dockerLoader.lastCaddyfile = caddyfile

	if caddyfileChanged {
		log.Info("New Caddyfile", zap.ByteString("caddyfile", caddyfile))

        tmpPath := CaddyfileAutosavePath + ".tmp"
        if err := os.WriteFile(tmpPath, caddyfile, 0640); err != nil {
            log.Warn("Failed to write temporary caddyfile", zap.Error(err), zap.String("path", tmpPath))
        } else if err := os.Rename(tmpPath, CaddyfileAutosavePath); err != nil {
            log.Warn("Failed to autosave caddyfile", zap.Error(err), zap.String("path", CaddyfileAutosavePath))
        }

		adapter := caddyconfig.GetAdapter("caddyfile")

		configJSON, warn, err := adapter.Adapt(caddyfile, nil)

		if warn != nil {
			log.Warn("Caddyfile to json warning", zap.String("warn", fmt.Sprintf("%v", warn)))
		}

		if err != nil {
			log.Error("Failed to convert caddyfile into json config", zap.Error(err))
			return false
		}

		log.Info("New Config JSON", zap.ByteString("json", configJSON))

		dockerLoader.lastJSONConfig = configJSON
		dockerLoader.lastVersion++
	}

	var wg sync.WaitGroup
	for _, server := range controlledServers {
		wg.Add(1)
		go dockerLoader.updateServer(&wg, server)
	}
	wg.Wait()

	return true
}

func (dockerLoader *DockerLoader) updateServer(wg *sync.WaitGroup, server string) {
	defer wg.Done()

	// Skip servers that are being updated already
	if dockerLoader.serversUpdating.Get(server) {
		return
	}

	// Flag and unflag updating
	dockerLoader.serversUpdating.Set(server, true)
	defer dockerLoader.serversUpdating.Delete(server)

	version := dockerLoader.lastVersion

	// Skip servers that already have this version
	if dockerLoader.serversVersions.Get(server) >= version {
		return
	}

	log := logger()
	log.Info("Sending configuration to", zap.String("server", server))

	url := "http://" + server + ":2019/load"

	postBody, err := addAdminListen(dockerLoader.lastJSONConfig, getServerAdminListen(dockerLoader.options, server))
	if err != nil {
		log.Error("Failed to add admin listen to", zap.String("server", server), zap.Error(err))
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(postBody))
	if err != nil {
		log.Error("Failed to create request to", zap.String("server", server), zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Error("Failed to send configuration to", zap.String("server", server), zap.Error(err))
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("Failed to read response from", zap.String("server", server), zap.Error(err))
		return
	}

	if resp.StatusCode != 200 {
		log.Error("Error response from server", zap.String("server", server), zap.Int("status code", resp.StatusCode), zap.ByteString("body", bodyBytes))
		return
	}

	dockerLoader.serversVersions.Set(server, version)

	log.Info("Successfully configured", zap.String("server", server))
}

func addAdminListen(configJSON []byte, listen string) ([]byte, error) {
	config := &caddy.Config{}
	err := json.Unmarshal(configJSON, config)
	if err != nil {
		return nil, err
	}
	// Respect explicit admin settings from Caddyfile/JSON config,
	// but override "admin off" since the plugin requires the admin API.
	if config.Admin != nil && !config.Admin.Disabled {
		return configJSON, nil
	}
	config.Admin = &caddy.AdminConfig{
		Listen: listen,
	}
	return json.Marshal(config)
}

func getServerAdminListen(options *config.Options, server string) string {
	if options.AdminListen != "" {
		return options.AdminListen
	}
	return "tcp/" + server + ":2019"
}
