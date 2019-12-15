module github.com/lucaslorentz/caddy-docker-proxy

go 1.12

replace github.com/h2non/gock => gopkg.in/h2non/gock.v1 v1.0.14

replace github.com/lucaslorentz/caddy-docker-proxy/plugin => ./plugin

require (
	github.com/caddyserver/caddy v1.0.3
	github.com/caddyserver/dnsproviders v0.3.0
	github.com/lucaslorentz/caddy-docker-proxy/plugin v0.0.0-00010101000000-000000000000
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/nicolasazrak/caddy-cache v0.3.4
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
	gotest.tools v2.2.0+incompatible // indirect
)
