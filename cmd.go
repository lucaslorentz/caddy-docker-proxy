package caddydockerproxy

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
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
				"Docker sockets comma separate.\n"+
					"Applicable to modes: controller, standalone")

			fs.String("docker-certs-path", "",
				"Docker socket certs path comma separate.\n"+
					"Applicable to modes: controller, standalone")

			fs.String("docker-apis-version", "",
				"Docker socket apis version comma separate.\n"+
					"Applicable to modes: controller, standalone")

			fs.String("controller-network", "",
				"Controller network name. Ex: caddy_controller.\n"+
					"When not defined, all networks attached to controller container are considered controller networks\n"+
					"Applicable to modes: controller, standalone")

			fs.String("controller-url", "",
				"Controller url, used by servers to fetch controller subnets. Ex: http://caddy-controller\n"+
					"Applicable to modes: server")

			fs.String("ingress-networks", "",
				"Comma separated name of ingress networks connecting caddy servers to containers.\n"+
					"When not defined, all networks attached to controller container are considered ingress networks\n"+
					"Applicable to modes: controller, standalone")

			fs.String("caddyfile-path", "",
				"Path to a base Caddyfile that will be extended with docker sites.\n"+
					"Applicable to modes: controller, standalone")

			fs.String("label-prefix", generator.DefaultLabelPrefix,
				"Prefix for Docker labels.\n"+
					"Applicable to modes: controller, standalone")

			fs.Bool("proxy-service-tasks", true,
				"Proxy to service tasks instead of service load balancer.\n"+
					"Applicable to modes: controller, standalone")

			fs.Bool("process-caddyfile", true,
				"Process Caddyfile before loading it, removing invalid servers.\n"+
					"Applicable to modes: controller, standalone")

			fs.Duration("polling-interval", 30*time.Second,
				"Interval caddy should manually check docker for a new caddyfile.\n"+
					"Applicable to modes: controller, standalone")

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

		bindAddress, err := getAdminListen(options)
		if err != nil {
			return 1, err
		}

		err = caddy.Run(&caddy.Config{
			Admin: &caddy.AdminConfig{
				Listen: bindAddress,
			},
		})
		if err != nil {
			return 1, err
		}
	}

	if options.Mode&config.Controller == config.Controller {
		log.Info("Running caddy proxy controller")
		controller := CreateCaddyController(options)
		if err := controller.Start(); err != nil {
			if err := caddy.Stop(); err != nil {
				return 1, err
			}

			return 1, err
		}
	}

	select {}
}

func getAdminListen(options *config.Options) (string, error) {
	if options.Mode&config.Controller == config.Controller {
		return "tcp/localhost:2019", nil
	}

	log := logger()

	var controllerNetworks []string

	if options.ControllerNetwork != "" {
		controllerNetworks = append(controllerNetworks, options.ControllerNetwork)
	}

	if options.ControllerUrl != nil {
		url := strings.TrimRight(options.ControllerUrl.String(), "/") + "/controller-subnets"
		log.Info("Fetching controller networks from url", zap.String("url", url))
		resp, err := http.Get(url)
		if err != nil {
			log.Error("Failed to fetch controller networks from contoller", zap.String("url", url), zap.Error(err))
			return "", err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		json.Unmarshal(body, &controllerNetworks)
	}

	var ipNets []*net.IPNet
	for _, controllerNetwork := range controllerNetworks {
		_, ipNet, err := net.ParseCIDR(controllerNetwork)
		if err != nil {
			log.Error("Failed to parse controller network", zap.String("ControllerNetwork", controllerNetwork), zap.Error(err))
			return "", err
		}
		ipNets = append(ipNets, ipNet)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		log.Error("Failed to get network interfaces", zap.Error(err))
		return "", err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			log.Error("Failed to get network interface addresses", zap.Error(err))
			return "", err
		}
		for _, addr := range addrs {
			for _, ipNet := range ipNets {
				switch v := addr.(type) {
				case *net.IPAddr:
					if ipNet.Contains(v.IP) {
						return "tcp/" + v.IP.String() + ":2019", nil
					}
				case *net.IPNet:
					if ipNet.Contains(v.IP) {
						return "tcp/" + v.IP.String() + ":2019", nil
					}
				}
			}
		}
	}

	return "tcp/0.0.0.0:2019", nil
}

func createOptions(flags caddycmd.Flags) *config.Options {
	caddyfilePath := flags.String("caddyfile-path")
	labelPrefixFlag := flags.String("label-prefix")
	proxyServiceTasksFlag := flags.Bool("proxy-service-tasks")
	processCaddyfileFlag := flags.Bool("process-caddyfile")
	pollingIntervalFlag := flags.Duration("polling-interval")
	modeFlag := flags.String("mode")
	controllerNetwork := flags.String("controller-network")
	controllerUrl := flags.String("controller-url")
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

	if controllerNetworkEnv := os.Getenv("CADDY_CONTROLLER_NETWORK"); controllerNetworkEnv != "" {
		options.ControllerNetwork = controllerNetworkEnv
	} else {
		options.ControllerNetwork = controllerNetwork
	}

	if controllerUrlEnv := os.Getenv("CADDY_CONTROLLER_URL"); controllerUrlEnv != "" {
		if url, err := url.Parse(controllerUrlEnv); err != nil {
			log.Error("Failed to parse CADDY_CONTROLLER_URL", zap.String("value", controllerUrlEnv), zap.Error(err))
		} else {
			options.ControllerUrl = url
		}
	} else if controllerUrl != "" {
		if url, err := url.Parse(controllerUrl); err != nil {
			log.Error("Failed to parse controller-url", zap.String("value", controllerUrl), zap.Error(err))
		} else {
			options.ControllerUrl = url
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

	return options
}
