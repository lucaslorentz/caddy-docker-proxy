module github.com/lucaslorentz/caddy-docker-proxy

go 1.13

replace github.com/lucaslorentz/caddy-docker-proxy/plugin => ./plugin

require (
	github.com/caddyserver/caddy v1.0.4
	github.com/caddyserver/dnsproviders v0.4.0
	github.com/lucaslorentz/caddy-docker-proxy/plugin v0.0.0-00010101000000-000000000000
	github.com/nicolasazrak/caddy-cache v0.3.4
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
)
