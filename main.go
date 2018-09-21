package main

import (
	// Plugins
	_ "github.com/lucaslorentz/caddy-docker-proxy/plugin"

	_ "github.com/caddyserver/dnsproviders/route53"

	// Caddy
	"github.com/mholt/caddy/caddy/caddymain"
)

func main() {
	caddymain.Run()

	// Keep caddy running after main instance is stopped
	select {}
}
