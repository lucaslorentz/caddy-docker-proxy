module github.com/lucaslorentz/caddy-docker-proxy

go 1.12

replace github.com/h2non/gock => gopkg.in/h2non/gock.v1 v1.0.14

replace github.com/lucaslorentz/caddy-docker-proxy/plugin => ./plugin

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/caddyserver/caddy v1.0.4
	github.com/caddyserver/dnsproviders v0.4.0
	github.com/lucaslorentz/caddy-docker-proxy/plugin v0.0.0-00010101000000-000000000000
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/nicolasazrak/caddy-cache v0.3.4
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
)
