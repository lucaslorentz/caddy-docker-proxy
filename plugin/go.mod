module github.com/lucaslorentz/caddy-docker-proxy/plugin

go 1.16

require (
	github.com/caddyserver/caddy/v2 v2.4.6
	github.com/containerd/containerd v1.5.7 // indirect
	github.com/docker/docker v20.10.10+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.19.0
)
