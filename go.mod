module github.com/lucaslorentz/caddy-docker-proxy/v2

go 1.14

replace github.com/lucaslorentz/caddy-docker-proxy/v2/plugin => ./plugin

require (
	github.com/caddyserver/caddy/v2 v2.0.0
	github.com/lucaslorentz/caddy-docker-proxy/v2/plugin v0.0.0-00010101000000-000000000000
)
