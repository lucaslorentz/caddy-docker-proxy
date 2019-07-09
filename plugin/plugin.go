package plugin

import (
	"github.com/caddyserver/caddy"
)

func init() {
	caddy.RegisterCaddyfileLoader("docker", CreateDockerLoader())
}
