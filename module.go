package caddydockerproxy

import "github.com/caddyserver/caddy/v2"

func init() {
	caddy.RegisterModule(CaddyDockerProxy{})
}

// Caddy docker proxy module
type CaddyDockerProxy struct {
}

// CaddyModule returns the Caddy module information.
func (CaddyDockerProxy) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "docker_proxy",
		New: func() caddy.Module { return new(CaddyDockerProxy) },
	}
}
