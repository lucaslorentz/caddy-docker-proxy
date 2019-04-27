package main

import (
	// Plugins
	_ "github.com/lucaslorentz/caddy-docker-proxy/plugin"

	// DNS Providers
	_ "github.com/caddyserver/dnsproviders/azure"
	_ "github.com/caddyserver/dnsproviders/cloudflare"
	_ "github.com/caddyserver/dnsproviders/digitalocean"
	_ "github.com/caddyserver/dnsproviders/godaddy"
	_ "github.com/caddyserver/dnsproviders/googlecloud"
	_ "github.com/caddyserver/dnsproviders/route53"

	// Plugins
	_ "github.com/nicolasazrak/caddy-cache"

	// Caddy
	"github.com/mholt/caddy/caddy/caddymain"
)

func main() {
	caddymain.Run()

	// Keep caddy running after main instance is stopped
	select {}
}
