package plugin

import (
	"flag"
	"log"
	"net"
	"os"
	"regexp"
	"time"

	"github.com/caddyserver/caddy/v2"
	caddycmd "github.com/caddyserver/caddy/v2/cmd"
	"github.com/lucaslorentz/caddy-docker-proxy/plugin/v2/config"
	"github.com/lucaslorentz/caddy-docker-proxy/plugin/v2/generator"
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
			fs.Bool("mode", false, "Which mode this instance should run: standalone | controller | server")
			fs.String("controller-network", "", "Network allowed to configure caddy server in CIDR notation. Ex: 10.200.200.0/24")
			fs.String("caddyfile-path", "", "Path to a base Caddyfile that will be extended with docker sites")
			fs.String("label-prefix", generator.DefaultLabelPrefix, "Prefix for Docker labels")
			fs.Bool("proxy-service-tasks", false, "Proxy to service tasks instead of service load balancer")
			fs.Bool("validate-network", true, "Validates if caddy container and target are in same network")
			fs.Bool("process-caddyfile", false, "Process Caddyfile before loading it, removing invalid servers")
			fs.Duration("polling-interval", 30*time.Second, "Interval caddy should manually check docker for a new caddyfile")
			return fs
		}(),
	})
}

func cmdFunc(flags caddycmd.Flags) (int, error) {
	caddy.TrapSignals()

	options := createOptions(flags)

	if options.Mode&config.Server == config.Server {
		log.Printf("[INFO] Running caddy proxy server")

		caddy.Run(&caddy.Config{
			Admin: &caddy.AdminConfig{
				Listen: getAdminListen(options),
			},
		})
	}

	if options.Mode&config.Controller == config.Controller {
		log.Printf("[INFO] Running caddy proxy controller")
		loader := CreateDockerLoader(options)
		loader.Start()
	}

	select {}
}

func getAdminListen(options *config.Options) string {
	if options.ControllerNetwork != nil {
		ifaces, err := net.Interfaces()
		if err != nil {
			log.Printf("[ERROR] Failed to get network interfaces: %v", err)
		}
		for _, i := range ifaces {
			addrs, err := i.Addrs()
			if err != nil {
				log.Printf("[ERROR] Failed to get network interface addresses: %v", err)
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
	labelPrefixFlag := flags.String("label-prefix")
	proxyServiceTasksFlag := flags.Bool("proxy-service-tasks")
	validateNetworkFlag := flags.Bool("validate-network")
	processCaddyfileFlag := flags.Bool("process-caddyfile")
	pollingIntervalFlag := flags.Duration("polling-interval")
	modeFlag := flags.String("mode")
	controllerSubnetFlag := flags.String("controller-network")

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

	if controllerIPRangeEnv := os.Getenv("CADDY_CONTROLLER_NETWORK"); controllerIPRangeEnv != "" {
		_, ipNet, err := net.ParseCIDR(controllerIPRangeEnv)
		if err != nil {
			log.Printf("[ERROR] Failed to parse CADDY_CONTROLLER_NETWORK %v: %v", controllerIPRangeEnv, err)
		} else if ipNet != nil {
			options.ControllerNetwork = ipNet
		}
	} else if controllerSubnetFlag != "" {
		_, ipNet, err := net.ParseCIDR(controllerSubnetFlag)
		if err != nil {
			log.Printf("[ERROR] Failed to parse controller-network %v: %v", controllerSubnetFlag, err)
		} else if ipNet != nil {
			options.ControllerNetwork = ipNet
		}
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

	if validateNetworkEnv := os.Getenv("CADDY_DOCKER_VALIDATE_NETWORK"); validateNetworkEnv != "" {
		options.ValidateNetwork = isTrue.MatchString(validateNetworkEnv)
	} else {
		options.ValidateNetwork = validateNetworkFlag
	}

	if processCaddyfileEnv := os.Getenv("CADDY_DOCKER_PROCESS_CADDYFILE"); processCaddyfileEnv != "" {
		options.ProcessCaddyfile = isTrue.MatchString(processCaddyfileEnv)
	} else {
		options.ProcessCaddyfile = processCaddyfileFlag
	}

	if pollingIntervalEnv := os.Getenv("CADDY_DOCKER_POLLING_INTERVAL"); pollingIntervalEnv != "" {
		if p, err := time.ParseDuration(pollingIntervalEnv); err != nil {
			log.Printf("[ERROR] Failed to parse CADDY_DOCKER_POLLING_INTERVAL: %v", err)
			options.PollingInterval = pollingIntervalFlag
		} else {
			options.PollingInterval = p
		}
	} else {
		options.PollingInterval = pollingIntervalFlag
	}

	return options
}
