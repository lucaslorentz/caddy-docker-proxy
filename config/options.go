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

	// SwarmMode enables atomic Caddyfile distribution via Swarm configs.
	// When enabled, this instance updates an existing Swarm service by mounting
	// a newly created, content-addressed Swarm config at SwarmCaddyfileTarget.
	SwarmMode            bool
	SwarmService         string
	SwarmCaddyfileTarget string
	SwarmConfigPrefix    string
	SwarmConfigHashLen   int
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
