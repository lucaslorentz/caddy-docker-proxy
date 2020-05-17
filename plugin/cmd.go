package plugin

import (
	"flag"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/caddyserver/caddy/v2"
	caddycmd "github.com/caddyserver/caddy/v2/cmd"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/plugin/config"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/plugin/generator"
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
			fs.String("label-prefix", generator.DefaultLabelPrefix, "Prefix for Docker labels")
			fs.String("caddyfile-path", "", "Path to a base CaddyFile that will be extended with docker sites")
			fs.Bool("proxy-service-tasks", false, "Proxy to service tasks instead of service load balancer")
			fs.Bool("validate-network", true, "Validates if caddy container and target are in same network")
			fs.Bool("process-caddyfile", false, "Process Caddyfile before loading it, removing invalid servers")
			fs.Duration("polling-interval", 30*time.Second, "Interval caddy should manually check docker for a new caddyfile")
			return fs
		}(),
	})
}

func cmdFunc(flags caddycmd.Flags) (int, error) {
	caddy.Load([]byte{}, true)
	options := createOptions(flags)
	loader := CreateDockerLoader(options)
	loader.Start()
	select {}
}

func createOptions(flags caddycmd.Flags) *config.Options {
	labelPrefixFlag := flags.String("label-prefix")
	caddyFilePath := flags.String("caddyfile-path")
	proxyServiceTasksFlag := flags.Bool("proxy-service-tasks")
	validateNetworkFlag := flags.Bool("validate-network")
	processCaddyfileFlag := flags.Bool("process-caddyfile")
	pollingIntervalFlag := flags.Duration("polling-interval")

	options := &config.Options{}

	if caddyFilePathEnv := os.Getenv("CADDY_DOCKER_CADDYFILE_PATH"); caddyFilePathEnv != "" {
		options.CaddyFilePath = caddyFilePathEnv
	} else {
		options.CaddyFilePath = caddyFilePath
	}

	if labelPrefixEnv := os.Getenv("CADDY_DOCKER_LABEL_PREFIX"); labelPrefixEnv != "" {
		options.LabelPrefix = labelPrefixEnv
	} else {
		options.LabelPrefix = labelPrefixFlag
	}

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
