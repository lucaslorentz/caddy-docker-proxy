package plugin

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// GenerateCaddyFile generates a caddy file config from docker swarm
func GenerateCaddyFile() []byte {
	var buffer bytes.Buffer

	client, err := client.NewEnvClient()
	if err != nil {
		addError(&buffer, err)
		return buffer.Bytes()
	}

	services, err := client.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		addError(&buffer, err)
		return buffer.Bytes()
	}

	for _, service := range services {
		addServiceToCaddyFile(&buffer, &service)
	}

	if buffer.Len() == 0 {
		buffer.WriteString("# Empty file")
	}

	return buffer.Bytes()
}

func addError(buffer *bytes.Buffer, e error) {
	for _, line := range strings.Split(e.Error(), `\n`) {
		buffer.WriteString(fmt.Sprintf("# %s", line))
	}
}

func addServiceToCaddyFile(buffer *bytes.Buffer, service *swarm.Service) {
	address := service.Spec.Labels["caddy.address"]
	if address == "" {
		return
	}

	targetPort := service.Spec.Labels["caddy.targetport"]
	targetPath := service.Spec.Labels["caddy.targetpath"]

	buffer.WriteString(fmt.Sprintf("%s {\n", address))
	buffer.WriteString(fmt.Sprintf("  proxy / %s:%s%s {\n", service.Spec.Name, targetPort, targetPath))
	buffer.WriteString("    transparent\n")

	if healthcheck, ok := service.Spec.Labels["caddy.healthcheck"]; ok {
		buffer.WriteString(fmt.Sprintf("    healthcheck %s\n", healthcheck))
	}

	if _, ok := service.Spec.Labels["caddy.websocket"]; ok {
		buffer.WriteString("    websocket\n")
	}

	buffer.WriteString("  }\n")
	buffer.WriteString("  gzip\n")

	if basicAuth, ok := service.Spec.Labels["caddy.basicauth"]; ok {
		buffer.WriteString(fmt.Sprintf("  basicauth / %s\n", basicAuth))
	}

	tls := service.Spec.Labels["caddy.tls"]
	if tlsDNS, ok := service.Spec.Labels["caddy.tls.dns"]; ok {
		buffer.WriteString(fmt.Sprintf("  tls %s {\n", tls))
		buffer.WriteString(fmt.Sprintf("    dns %s\n", tlsDNS))
		buffer.WriteString("  }\n")
	} else if tls != "" {
		buffer.WriteString(fmt.Sprintf("  tls %s\n", tls))
	}

	buffer.WriteString("}\n")
}
