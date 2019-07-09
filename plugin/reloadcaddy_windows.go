package plugin

import (
	"log"
	"time"

	"github.com/caddyserver/caddy"

	httpserver "github.com/caddyserver/caddy/caddyhttp/httpserver"
)

// ReloadCaddy reloads caddy
func ReloadCaddy(loader caddy.Loader) {
	httpserver.GracefulTimeout = 20 * time.Second

	log.Printf("[INFO] Reloading\n")

	instances := caddy.Instances()

	for _, instance := range instances {
		log.Printf("[INFO] Stopping current instance")

		instance.ShutdownCallbacks()

		err := instance.Stop()
		if err != nil {
			log.Printf("[ERROR] %v", err.Error())
			log.Fatal(err)
		}

		input, err := loader.Load("http")
		if err != nil {
			log.Printf("[ERROR] %v", err.Error())
			log.Fatal(err)
		}

		log.Printf("[INFO] Starting new instance")
		instance, err = caddy.Start(input)
		if err != nil {
			log.Printf("[ERROR] %v", err.Error())
			log.Fatal(err)
		}
	}
}
