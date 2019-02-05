package main

import (
	_ "github.com/caddyserver/dnsproviders/duckdns"
	_ "github.com/lucaslorentz/caddy-docker-proxy/plugin"
	"github.com/mholt/caddy/caddy/caddymain"
)

func main() {
	caddymain.EnableTelemetry = false
	caddymain.Run()

	// Keep caddy running after main instance is stopped
	select {}
}
