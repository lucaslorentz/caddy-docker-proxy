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
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
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
	dockerClients   []docker.Client
	generator       *generator.CaddyfileGenerator
	timer           *time.Timer
	skipEvents      []bool
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

	dockerClients := []docker.Client{}
	for i, dockerSocket := range dockerLoader.options.DockerSockets {
		// cf https://github.com/docker/go-docker/blob/master/client.go
		// setenv to use NewEnvClient
		// or manually

		os.Setenv("DOCKER_HOST", dockerSocket)

		if len(dockerLoader.options.DockerCertsPath) >= i+1 && dockerLoader.options.DockerCertsPath[i] != "" {
			os.Setenv("DOCKER_CERT_PATH", dockerLoader.options.DockerCertsPath[i])
		} else {
			os.Unsetenv("DOCKER_CERT_PATH")
		}

		if len(dockerLoader.options.DockerAPIsVersion) >= i+1 && dockerLoader.options.DockerAPIsVersion[i] != "" {
			os.Setenv("DOCKER_API_VERSION", dockerLoader.options.DockerAPIsVersion[i])
		} else {
			os.Unsetenv("DOCKER_API_VERSION")
		}

		dockerClient, err := client.NewEnvClient()
		if err != nil {
			log.Error("Docker connection failed to docker specify socket", zap.Error(err), zap.String("DockerSocket", dockerSocket))
			return err
		}

		dockerPing, err := dockerClient.Ping(context.Background())
		if err != nil {
			log.Error("Docker ping failed on specify socket", zap.Error(err), zap.String("DockerSocket", dockerSocket))
			return err
		}

		dockerClient.NegotiateAPIVersionPing(dockerPing)

		wrappedClient := docker.WrapClient(dockerClient)

		dockerClients = append(dockerClients, wrappedClient)
	}

	// by default it will used the env docker
	if len(dockerClients) == 0 {
		dockerClient, err := client.NewEnvClient()
		dockerLoader.options.DockerSockets = append(dockerLoader.options.DockerSockets, os.Getenv("DOCKER_HOST"))
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

		dockerClients = append(dockerClients, wrappedClient)
	}

	dockerLoader.dockerClients = dockerClients
	dockerLoader.skipEvents = make([]bool, len(dockerLoader.dockerClients))

	dockerLoader.generator = generator.CreateGenerator(
		dockerClients,
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
	)

	ready := make(chan struct{})
	dockerLoader.timer = time.AfterFunc(0, func() {
		<-ready
		dockerLoader.update()
	})
	close(ready)

	go dockerLoader.monitorEvents()

	return nil
}

func (dockerLoader *DockerLoader) monitorEvents() {
	for {
		dockerLoader.listenEvents()
		time.Sleep(30 * time.Second)
	}
}

func (dockerLoader *DockerLoader) listenEvents() {
	args := filters.NewArgs()
	if !isTrue.MatchString(os.Getenv("CADDY_DOCKER_NO_SCOPE")) {
		// This env var is useful for Podman where in some instances the scope can cause some issues.
		args.Add("scope", "swarm")
		args.Add("scope", "local")
	}
	args.Add("type", "service")
	args.Add("type", "container")
	args.Add("type", "config")
	args.Add("type", "network")

	for i, dockerClient := range dockerLoader.dockerClients {
		context, cancel := context.WithCancel(context.Background())

		eventsChan, errorChan := dockerClient.Events(context, types.EventsOptions{
			Filters: args,
		})

		log := logger()
		log.Info("Connecting to docker events", zap.String("DockerSocket", dockerLoader.options.DockerSockets[i]))

	ListenEvents:
		for {
			select {
			case event := <-eventsChan:
				if dockerLoader.skipEvents[i] {
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
					dockerLoader.skipEvents[i] = true
					dockerLoader.timer.Reset(dockerLoader.options.EventThrottleInterval)
				}
			case err := <-errorChan:
				cancel()
				if err != nil {
					log.Error("Docker events error", zap.Error(err))
				}
				break ListenEvents
			}
		}
	}
}

func (dockerLoader *DockerLoader) update() bool {
	dockerLoader.timer.Reset(dockerLoader.options.PollingInterval)
	for i := 0; i < len(dockerLoader.skipEvents); i++ {
		dockerLoader.skipEvents[i] = false
	}

	// Don't cache the logger more globally, it can change based on config reloads
	log := logger()
	caddyfile, controlledServers := dockerLoader.generator.GenerateCaddyfile(log)

	caddyfileChanged := !bytes.Equal(dockerLoader.lastCaddyfile, caddyfile)

	dockerLoader.lastCaddyfile = caddyfile

	if caddyfileChanged {
		log.Info("New Caddyfile", zap.ByteString("caddyfile", caddyfile))

		if autosaveErr := os.WriteFile(CaddyfileAutosavePath, caddyfile, 0666); autosaveErr != nil {
			log.Warn("Failed to autosave caddyfile", zap.Error(autosaveErr), zap.String("path", CaddyfileAutosavePath))
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

	postBody, err := addAdminListen(dockerLoader.lastJSONConfig, "tcp/"+server+":2019")
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
	config.Admin = &caddy.AdminConfig{
		Listen: listen,
	}
	return json.Marshal(config)
}
