package main

import (
	// Plugins
	_ "github.com/lucaslorentz/caddy-docker-proxy/plugin"

	_ "github.com/caddyserver/dnsproviders/route53"

	// Caddy
	"github.com/mholt/caddy/caddy/caddymain"
)

// DisableTelemetryFlag if set, it will disable telemetry that Caddy enables by default
var DisableTelemetryFlag string

func main() {
	if DisableTelemetryFlag != "" {
		caddymain.EnableTelemetry = false
	}

	caddymain.Run()

	// Keep caddy running after main instance is stopped
	select {}
}
