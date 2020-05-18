package main

import (
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	// plug in Caddy modules here
	_ "github.com/caddyserver/caddy/v2/modules/standard"
	_ "github.com/lucaslorentz/caddy-docker-proxy/v2/plugin"
)

func main() {
	caddycmd.Main()
}
