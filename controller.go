package caddydockerproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/docker"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/generator"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/utils"

	"go.uber.org/zap"
)

var CaddyfileAutosavePath = filepath.Join(caddy.AppConfigDir(), "Caddyfile.autosave")

// CaddyController generates caddy files from docker swarm information and send to caddy servers
type CaddyController struct {
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

// CreateCaddyController creates a caddy controller
func CreateCaddyController(options *config.Options) *CaddyController {
	return &CaddyController{
		options:         options,
		serversVersions: utils.NewStringInt64CMap(),
		serversUpdating: utils.NewStringBoolCMap(),
	}
}

func logger() *zap.Logger {
	return caddy.Log().
		Named("docker-proxy")
}

// Start controller
func (controller *CaddyController) Start() error {
	if !controller.initialized {
		controller.initialized = true
		log := logger()

		dockerClients := []docker.Client{}
		for i, dockerSocket := range controller.options.DockerSockets {
			// cf https://github.com/docker/go-docker/blob/master/client.go
			// setenv to use NewEnvClient
			// or manually

			os.Setenv("DOCKER_HOST", dockerSocket)

			if len(controller.options.DockerCertsPath) >= i+1 && controller.options.DockerCertsPath[i] != "" {
				os.Setenv("DOCKER_CERT_PATH", controller.options.DockerCertsPath[i])
			} else {
				os.Unsetenv("DOCKER_CERT_PATH")
			}

			if len(controller.options.DockerAPIsVersion) >= i+1 && controller.options.DockerAPIsVersion[i] != "" {
				os.Setenv("DOCKER_API_VERSION", controller.options.DockerAPIsVersion[i])
			} else {
				os.Unsetenv("DOCKER_API_VERSION")
			}

			dockerClient, err := client.NewClientWithOpts(client.FromEnv)
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
			dockerClient, err := client.NewClientWithOpts(client.FromEnv)
			controller.options.DockerSockets = append(controller.options.DockerSockets, os.Getenv("DOCKER_HOST"))
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

		controller.dockerClients = dockerClients
		controller.skipEvents = make([]bool, len(controller.dockerClients))

		controller.generator = generator.CreateGenerator(
			dockerClients,
			docker.CreateUtils(),
			controller.options,
		)

		log.Info("Start", zap.Any("options", controller.options))

		ready := make(chan struct{})
		controller.timer = time.AfterFunc(0, func() {
			<-ready
			controller.update()
		})
		close(ready)

		if controller.options.Mode&config.Server == 0 {
			err := controller.startHttpServer()
			if err != nil {
				return err
			}
		}

		go controller.monitorEvents()
	}

	return nil
}

func (controller *CaddyController) startHttpServer() error {
	http.HandleFunc("/controller-subnets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		controllerNetworkGroup, err := controller.generator.GetControllerNetworkGroup(logger())
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var controllerSubnets []string
		for _, networkInfo := range controllerNetworkGroup.Networks {
			for _, subnet := range networkInfo.Subnets {
				controllerSubnets = append(controllerSubnets, subnet.String())
			}
		}

		err = json.NewEncoder(w).Encode(controllerSubnets)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	})
	return http.ListenAndServe(":80", nil)
}

func (controller *CaddyController) monitorEvents() {
	for {
		controller.listenEvents()
		time.Sleep(30 * time.Second)
	}
}

func (controller *CaddyController) listenEvents() {
	args := filters.NewArgs()
	if !isTrue.MatchString(os.Getenv("CADDY_DOCKER_NO_SCOPE")) {
		// This env var is useful for Podman where in some instances the scope can cause some issues.
		args.Add("scope", "swarm")
		args.Add("scope", "local")
	}
	args.Add("type", "service")
	args.Add("type", "container")
	args.Add("type", "config")

	for i, dockerClient := range controller.dockerClients {
		context, cancel := context.WithCancel(context.Background())

		eventsChan, errorChan := dockerClient.Events(context, types.EventsOptions{
			Filters: args,
		})

		log := logger()
		log.Info("Connecting to docker events", zap.String("DockerSocket", controller.options.DockerSockets[i]))

	ListenEvents:
		for {
			select {
			case event := <-eventsChan:
				if controller.skipEvents[i] {
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
					(event.Type == "config" && event.Action == "remove")

				if update {
					controller.skipEvents[i] = true
					controller.timer.Reset(100 * time.Millisecond)
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

func (controller *CaddyController) update() bool {
	controller.timer.Reset(controller.options.PollingInterval)
	for i := 0; i < len(controller.skipEvents); i++ {
		controller.skipEvents[i] = false
	}

	// Don't cache the logger more globally, it can change based on config reloads
	log := logger()
	caddyfile, controlledServers := controller.generator.GenerateCaddyfile(log)

	caddyfileChanged := !bytes.Equal(controller.lastCaddyfile, caddyfile)

	controller.lastCaddyfile = caddyfile

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

		controller.lastJSONConfig = configJSON
		controller.lastVersion++
	}

	var wg sync.WaitGroup
	for _, server := range controlledServers {
		wg.Add(1)
		go controller.updateServer(&wg, server)
	}
	wg.Wait()

	return true
}

func (controller *CaddyController) updateServer(wg *sync.WaitGroup, server string) {
	defer wg.Done()

	// Skip servers that are being updated already
	if controller.serversUpdating.Get(server) {
		return
	}

	// Flag and unflag updating
	controller.serversUpdating.Set(server, true)
	defer controller.serversUpdating.Delete(server)

	version := controller.lastVersion

	// Skip servers that already have this version
	if controller.serversVersions.Get(server) >= version {
		return
	}

	log := logger()
	log.Info("Sending configuration to", zap.String("server", server))

	url := "http://" + server + ":2019/load"

	postBody, err := addAdminListen(controller.lastJSONConfig, "tcp/"+server+":2019")
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

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("Failed to read response from", zap.String("server", server), zap.Error(err))
		return
	}

	if resp.StatusCode != 200 {
		log.Error("Error response from server", zap.String("server", server), zap.Int("status code", resp.StatusCode), zap.ByteString("body", bodyBytes))
		return
	}

	controller.serversVersions.Set(server, version)

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
