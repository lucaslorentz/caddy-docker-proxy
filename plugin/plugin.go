package plugin

import (
	"github.com/mholt/caddy"
)

func init() {
	caddy.SetDefaultCaddyfileLoader("docker", CreateDockerLoader())
}
