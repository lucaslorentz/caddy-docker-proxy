package caddydockerproxy

import (
	"flag"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	caddycmd "github.com/caddyserver/caddy/v2/cmd"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/config"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/generator"

	"go.uber.org/zap"
)

var isTrue = regexp.MustCompile("(?i)^(true|yes|1)$")

func init() {
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "docker-proxy",
		Func:  cmdFunc,
		Usage: "<command>",
		Short: "Run caddy as a docker proxy",
		Flags: func() *flag.FlagSet {
			fs := flag.NewFlagSet("docker-proxy", flag.ExitOnError)

			fs.Bool("mode", false,
				"Which mode this instance should run: standalone | controller | server")

			fs.String("docker-sockets", "",
				"Docker sockets comma separate")

			fs.String("docker-certs-path", "",
				"Docker socket certs path comma separate")

			fs.String("docker-apis-version", "",
				"Docker socket apis version comma separate")

			fs.String("controller-network", "",
				"Network allowed to configure caddy server in CIDR notation. Ex: 10.200.200.0/24")

			fs.String("ingress-networks", "",
				"Comma separated name of ingress networks connecting caddy servers to containers.\n"+
					"When not defined, networks attached to controller container are considered ingress networks")

			fs.String("caddyfile-path", "",
				"Path to a base Caddyfile that will be extended with docker sites")

			fs.String("envfile", "",
				"Environment file with environment variables in the KEY=VALUE format")

			fs.String("label-prefix", generator.DefaultLabelPrefix,
				"Prefix for Docker labels")

			fs.Bool("proxy-service-tasks", true,
				"Proxy to service tasks instead of service load balancer")

			fs.Bool("process-caddyfile", true,
				"Process Caddyfile before loading it, removing invalid servers")

			fs.Bool("scan-stopped-containers", true,
				"Scan stopped containers and use its labels for caddyfile generation")

			fs.Duration("polling-interval", 30*time.Second,
				"Interval caddy should manually check docker for a new caddyfile")

			fs.Duration("event-throttle-interval", 100*time.Millisecond,
				"Interval to throttle caddyfile updates triggered by docker events")

			return fs
		}(),
	})
}

func cmdFunc(flags caddycmd.Flags) (int, error) {
	caddy.TrapSignals()

	options := createOptions(flags)
	log := logger()

	if options.Mode&config.Server == config.Server {
		log.Info("Running caddy proxy server")

		err := caddy.Run(&caddy.Config{
			Admin: &caddy.AdminConfig{
				Listen: getAdminListen(options),
			},
		})
		if err != nil {
			return 1, err
		}
	}

	if options.Mode&config.Controller == config.Controller {
		log.Info("Running caddy proxy controller")
		loader := CreateDockerLoader(options)
		if err := loader.Start(); err != nil {
			if err := caddy.Stop(); err != nil {
				return 1, err
			}

			return 1, err
		}
	}

	select {}
}

func getAdminListen(options *config.Options) string {
	if options.ControllerNetwork != nil {
		ifaces, err := net.Interfaces()
		log := logger()

		if err != nil {
			log.Error("Failed to get network interfaces", zap.Error(err))
		}
		for _, i := range ifaces {
			addrs, err := i.Addrs()
			if err != nil {
				log.Error("Failed to get network interface addresses", zap.Error(err))
				continue
			}
			for _, a := range addrs {
				switch v := a.(type) {
				case *net.IPAddr:
					if options.ControllerNetwork.Contains(v.IP) {
						return "tcp/" + v.IP.String() + ":2019"
					}
					break
				case *net.IPNet:
					if options.ControllerNetwork.Contains(v.IP) {
						return "tcp/" + v.IP.String() + ":2019"
					}
					break
				}
			}
		}
	}
	return "tcp/localhost:2019"
}

func createOptions(flags caddycmd.Flags) *config.Options {
	caddyfilePath := flags.String("caddyfile-path")
	envFile := flags.String("envfile")
	labelPrefixFlag := flags.String("label-prefix")
	proxyServiceTasksFlag := flags.Bool("proxy-service-tasks")
	processCaddyfileFlag := flags.Bool("process-caddyfile")
	scanStoppedContainersFlag := flags.Bool("scan-stopped-containers")
	pollingIntervalFlag := flags.Duration("polling-interval")
	eventThrottleIntervalFlag := flags.Duration("event-throttle-interval")
	modeFlag := flags.String("mode")
	controllerSubnetFlag := flags.String("controller-network")
	dockerSocketsFlag := flags.String("docker-sockets")
	dockerCertsPathFlag := flags.String("docker-certs-path")
	dockerAPIsVersionFlag := flags.String("docker-apis-version")
	ingressNetworksFlag := flags.String("ingress-networks")

	options := &config.Options{}

	var mode string
	if modeEnv := os.Getenv("CADDY_DOCKER_MODE"); modeEnv != "" {
		mode = modeEnv
	} else {
		mode = modeFlag
	}
	switch mode {
	case "controller":
		options.Mode = config.Controller
		break
	case "server":
		options.Mode = config.Server
	default:
		options.Mode = config.Standalone
	}

	log := logger()

	if dockerSocketsEnv := os.Getenv("CADDY_DOCKER_SOCKETS"); dockerSocketsEnv != "" {
		options.DockerSockets = strings.Split(dockerSocketsEnv, ",")
	} else if dockerSocketsFlag != "" {
		options.DockerSockets = strings.Split(dockerSocketsFlag, ",")
	} else {
		options.DockerSockets = nil
	}

	if dockerCertsPathEnv := os.Getenv("CADDY_DOCKER_CERTS_PATH"); dockerCertsPathEnv != "" {
		options.DockerCertsPath = strings.Split(dockerCertsPathEnv, ",")
	} else {
		options.DockerCertsPath = strings.Split(dockerCertsPathFlag, ",")
	}

	if dockerAPIsVersionEnv := os.Getenv("CADDY_DOCKER_APIS_VERSION"); dockerAPIsVersionEnv != "" {
		options.DockerAPIsVersion = strings.Split(dockerAPIsVersionEnv, ",")
	} else {
		options.DockerAPIsVersion = strings.Split(dockerAPIsVersionFlag, ",")
	}

	if controllerIPRangeEnv := os.Getenv("CADDY_CONTROLLER_NETWORK"); controllerIPRangeEnv != "" {
		_, ipNet, err := net.ParseCIDR(controllerIPRangeEnv)
		if err != nil {
			log.Error("Failed to parse CADDY_CONTROLLER_NETWORK", zap.String("CADDY_CONTROLLER_NETWORK", controllerIPRangeEnv), zap.Error(err))
		} else if ipNet != nil {
			options.ControllerNetwork = ipNet
		}
	} else if controllerSubnetFlag != "" {
		_, ipNet, err := net.ParseCIDR(controllerSubnetFlag)
		if err != nil {
			log.Error("Failed to parse controller-network", zap.String("controller-network", controllerSubnetFlag), zap.Error(err))
		} else if ipNet != nil {
			options.ControllerNetwork = ipNet
		}
	}

	if ingressNetworksEnv := os.Getenv("CADDY_INGRESS_NETWORKS"); ingressNetworksEnv != "" {
		options.IngressNetworks = strings.Split(ingressNetworksEnv, ",")
	} else if ingressNetworksFlag != "" {
		options.IngressNetworks = strings.Split(ingressNetworksFlag, ",")
	}

	if caddyfilePathEnv := os.Getenv("CADDY_DOCKER_CADDYFILE_PATH"); caddyfilePathEnv != "" {
		options.CaddyfilePath = caddyfilePathEnv
	} else {
		options.CaddyfilePath = caddyfilePath
	}

	if envFileEnv := os.Getenv("CADDY_DOCKER_ENVFILE"); envFileEnv != "" {
		options.EnvFile = envFileEnv
	} else {
		options.EnvFile = envFile
	}

	if labelPrefixEnv := os.Getenv("CADDY_DOCKER_LABEL_PREFIX"); labelPrefixEnv != "" {
		options.LabelPrefix = labelPrefixEnv
	} else {
		options.LabelPrefix = labelPrefixFlag
	}
	options.ControlledServersLabel = options.LabelPrefix + "_controlled_server"

	if proxyServiceTasksEnv := os.Getenv("CADDY_DOCKER_PROXY_SERVICE_TASKS"); proxyServiceTasksEnv != "" {
		options.ProxyServiceTasks = isTrue.MatchString(proxyServiceTasksEnv)
	} else {
		options.ProxyServiceTasks = proxyServiceTasksFlag
	}

	if processCaddyfileEnv := os.Getenv("CADDY_DOCKER_PROCESS_CADDYFILE"); processCaddyfileEnv != "" {
		options.ProcessCaddyfile = isTrue.MatchString(processCaddyfileEnv)
	} else {
		options.ProcessCaddyfile = processCaddyfileFlag
	}

	if scanStoppedContainersEnv := os.Getenv("CADDY_DOCKER_SCAN_STOPPED_CONTAINERS"); scanStoppedContainersEnv != "" {
		options.ScanStoppedContainers = isTrue.MatchString(scanStoppedContainersEnv)
	} else {
		options.ScanStoppedContainers = scanStoppedContainersFlag
	}

	if pollingIntervalEnv := os.Getenv("CADDY_DOCKER_POLLING_INTERVAL"); pollingIntervalEnv != "" {
		if p, err := time.ParseDuration(pollingIntervalEnv); err != nil {
			log.Error("Failed to parse CADDY_DOCKER_POLLING_INTERVAL", zap.String("CADDY_DOCKER_POLLING_INTERVAL", pollingIntervalEnv), zap.Error(err))
			options.PollingInterval = pollingIntervalFlag
		} else {
			options.PollingInterval = p
		}
	} else {
		options.PollingInterval = pollingIntervalFlag
	}

	if eventThrottleIntervalEnv := os.Getenv("CADDY_DOCKER_EVENT_THROTTLE_INTERVAL"); eventThrottleIntervalEnv != "" {
		if p, err := time.ParseDuration(eventThrottleIntervalEnv); err != nil {
			log.Error("Failed to parse CADDY_DOCKER_EVENT_THROTTLE_INTERVAL", zap.String("CADDY_DOCKER_EVENT_THROTTLE_INTERVAL", eventThrottleIntervalEnv), zap.Error(err))
			options.EventThrottleInterval = pollingIntervalFlag
		} else {
			options.EventThrottleInterval = p
		}
	} else {
		options.EventThrottleInterval = eventThrottleIntervalFlag
	}

	return options
}
