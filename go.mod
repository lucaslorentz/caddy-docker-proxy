module github.com/lucaslorentz/caddy-docker-proxy

go 1.12

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/caddyserver/caddy v1.0.3
	github.com/caddyserver/dnsproviders v0.3.0
	github.com/containerd/containerd v1.2.7 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.7.3-0.20190816182709-c9aee96bfd1b
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/morikuni/aec v0.0.0-20170113033406-39771216ff4c // indirect
	github.com/nicolasazrak/caddy-cache v0.3.4
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
	github.com/stretchr/testify v1.4.0
	gotest.tools v2.2.0+incompatible // indirect
)

replace github.com/h2non/gock => gopkg.in/h2non/gock.v1 v1.0.14
