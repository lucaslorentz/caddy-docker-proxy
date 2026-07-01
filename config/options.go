package config

import (
	"net"
	"time"
)

// Options are the options for generator
type Options struct {
	CaddyfilePath          string
	EnvFile                string
	AdminListen            string
	AdminDisabled          bool
	DockerSockets          []string
	DockerCertsPath        []string
	DockerAPIsVersion      []string
	LabelPrefix            string
	ControlledServersLabel string
	ProxyServiceTasks      bool
	ProcessCaddyfile       bool
	ScanStoppedContainers  bool
	PollingInterval        time.Duration
	EventThrottleInterval  time.Duration
	Mode                   Mode
	Secret                 string
	ControllerNetwork      *net.IPNet
	IngressNetworks        []string

	// LogLevel and LogFormat configure Caddy's logging (level and encoder).
	// They apply in all modes — including controller mode, via a minimal
	// logging-only Caddy instance — and are re-applied on every pushed config
	// so they survive reloads. Empty values keep Caddy's defaults.
	LogLevel  string
	LogFormat string
}

// Mode represents how this instance should run
type Mode int

const (
	// Controller runs only controller
	Controller Mode = 1
	// Server runs only server
	Server Mode = 2
	// Standalone runs controller and server in a single instance
	Standalone Mode = Controller | Server
)
