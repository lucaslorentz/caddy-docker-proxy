// +build !windows

package plugin

import (
	"os"
	"syscall"

	"github.com/caddyserver/caddy"
)

func ReloadCaddy(loader caddy.Loader) {
	self, _ := os.FindProcess(os.Getpid())
	self.Signal(syscall.SIGUSR1)
}
