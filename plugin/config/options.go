package config

import (
	"net"
	"time"
)

// Options are the options for generator
type Options struct {
	CaddyfilePath          string
	LabelPrefix            string
	ControlledServersLabel string
	ProxyServiceTasks      bool
	ProcessCaddyfile       bool
	PollingInterval        time.Duration
	Mode                   Mode
	Secret                 string
	ControllerNetwork      *net.IPNet
	IngressNetworks        []string
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
