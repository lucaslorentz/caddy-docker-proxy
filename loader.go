package caddydockerproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"os"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/joho/godotenv"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/docker"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/generator"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/utils"
	"github.com/moby/moby/client"

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
	caddyLogging    *caddy.Logging
}

// CreateDockerLoader creates a docker loader
func CreateDockerLoader(options *config.Options) *DockerLoader {
	return &DockerLoader{
		options:         options,
		serversVersions: utils.NewStringInt64CMap(),
		serversUpdating: utils.NewStringBoolCMap(),
		caddyLogging:    buildCaddyLoggingConfig(options),
	}
}

func logger() *zap.Logger {
	return caddy.Log().Named("docker-proxy")
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
		// set env to configure the Docker client
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

		dockerClient, err := client.New(client.FromEnv)
		if err != nil {
			log.Error("Docker connection failed to docker specify socket", zap.Error(err), zap.String("DockerSocket", dockerSocket))
			return err
		}

		_, err = dockerClient.Ping(context.Background(), client.PingOptions{NegotiateAPIVersion: true})
		if err != nil {
			log.Error("Docker ping failed on specify socket", zap.Error(err), zap.String("DockerSocket", dockerSocket))
			return err
		}

		wrappedClient := docker.WrapClient(dockerClient)

		dockerClients = append(dockerClients, wrappedClient)
	}

	// by default it will used the env docker
	if len(dockerClients) == 0 {
		dockerClient, err := client.New(client.FromEnv)
		dockerLoader.options.DockerSockets = append(dockerLoader.options.DockerSockets, os.Getenv("DOCKER_HOST"))
		if err != nil {
			log.Error("Docker connection failed", zap.Error(err))
			return err
		}

		_, err = dockerClient.Ping(context.Background(), client.PingOptions{NegotiateAPIVersion: true})
		if err != nil {
			log.Error("Docker ping failed", zap.Error(err))
			return err
		}

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
		zap.String("CaddyfileAutosavePath", CaddyfileAutosavePath),
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
	args := make(client.Filters)
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

		eventsChan, errorChan := dockerClient.Events(context, client.EventsListOptions{
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
		log.Debug("New Caddyfile", zap.ByteString("caddyfile", caddyfile))

		tmpPath := CaddyfileAutosavePath + ".tmp"
		if err := os.MkdirAll(filepath.Dir(CaddyfileAutosavePath), 0700); err != nil {
			log.Warn("Failed to create autosave directory", zap.Error(err), zap.String("path", filepath.Dir(CaddyfileAutosavePath)))
		} else if err := os.WriteFile(tmpPath, caddyfile, 0640); err != nil {
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

		log.Debug("New Config JSON", zap.ByteString("json", configJSON))

		dockerLoader.lastJSONConfig = configJSON
		dockerLoader.lastVersion++
	}

	var wg sync.WaitGroup
	for _, server := range controlledServers {
		wg.Add(1)
		go dockerLoader.updateServer(&wg, server)
	}
	// When this instance also serves (standalone/server mode), push to the
	// in-process Caddy as well. The generator lists only remote servers, so the
	// local target is added here.
	if dockerLoader.options.Mode&config.Server == config.Server {
		wg.Add(1)
		go dockerLoader.updateServer(&wg, localServer)
	}
	wg.Wait()

	return true
}

// localServer is the controlledServers entry that represents this in-process
// Caddy. It is pushed via caddy.Load instead of the admin API.
const localServer = "localhost"

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

	postBody, err := dockerLoader.prepareServerConfig(server)
	if err != nil {
		log.Error("Failed to prepare configuration for", zap.String("server", server), zap.Error(err))
		return
	}

	// The local target loads in-process; remote targets POST to the admin API.
	if server == localServer {
		err = pushLocal(postBody)
	} else {
		err = pushRemoteAdmin(server, postBody)
	}
	if err != nil {
		log.Error("Failed to send configuration to", zap.String("server", server), zap.Error(err))
		return
	}

	dockerLoader.serversVersions.Set(server, version)

	log.Info("Successfully configured", zap.String("server", server))
}

// prepareServerConfig builds the config to push to server from the loader's last
// generated config, with a single unmarshal/marshal round-trip.
func (dockerLoader *DockerLoader) prepareServerConfig(server string) ([]byte, error) {
	config := &caddy.Config{}
	if err := json.Unmarshal(dockerLoader.lastJSONConfig, config); err != nil {
		return nil, err
	}

	// Remote servers always need a reachable admin endpoint for controller
	// pushes, so an absent or disabled admin config falls back to the server
	// address. The local instance loads in-process and leaves the decision to
	// the config: an explicit admin config ("admin off" included) is respected,
	// and an absent one gets Caddy's own default. CADDY_ADMIN supplies the
	// local listen address or "off" - applied here because Caddy itself has no
	// disable semantics for that variable.
	if server == localServer {
		if config.Admin == nil {
			if dockerLoader.options.AdminDisabled {
				config.Admin = &caddy.AdminConfig{Disabled: true}
			} else if dockerLoader.options.AdminListen != "" {
				config.Admin = &caddy.AdminConfig{Listen: dockerLoader.options.AdminListen}
			} else {
				config.Admin = &caddy.AdminConfig{Listen: defaultAdminListen}
			}
		}
	} else if config.Admin == nil || config.Admin.Disabled {
		config.Admin = &caddy.AdminConfig{Listen: getServerAdminListen(dockerLoader.options, server)}
	}

	// Re-apply our logging only to the local Caddy (the standalone self-push),
	// so --log-level/--log-format survive its config reload. Remote servers run
	// their own instance and manage their own logging. Logging already defined
	// in the config (e.g. via labels) is respected.
	if server == localServer && dockerLoader.caddyLogging != nil && (config.Logging == nil || len(config.Logging.Logs) == 0) {
		config.Logging = dockerLoader.caddyLogging
	}

	return json.Marshal(config)
}

func getServerAdminListen(options *config.Options, server string) string {
	if options.AdminListen != "" {
		return options.AdminListen
	}
	return "tcp/" + server + ":2019"
}
