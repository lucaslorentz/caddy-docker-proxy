package main

import (
	"flag"
	"os"
	"regexp"

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

var enableTelemetryFlag bool
var isTrue = regexp.MustCompile("(?i)^(true|yes|1)$")

func main() {
	flag.BoolVar(&enableTelemetryFlag, "enable-telemetry", false, "Enable caddy telemetry")

	flag.Parse()

	if enableTelemetryEnv := os.Getenv("CADDY_ENABLE_TELEMETRY"); enableTelemetryEnv != "" {
		caddymain.EnableTelemetry = isTrue.MatchString(enableTelemetryEnv)
	} else {
		caddymain.EnableTelemetry = enableTelemetryFlag
	}

	caddymain.Run()

	// Keep caddy running after main instance is stopped
	select {}
}
