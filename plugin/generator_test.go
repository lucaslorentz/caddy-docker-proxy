package plugin

import (
	"bytes"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/assert"
)

func TestAddServiceAutoMapping(t *testing.T) {
	var service = swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "service",
				Labels: map[string]string{
					"caddy":                   "{{.Spec.Name}}.testdomain.com",
					"caddy.proxy":             "/ {{.Spec.Name}}:5000/api",
					"caddy.proxy.transparent": "",
					"caddy.proxy.healthcheck": "/health",
					"caddy.proxy.websocket":   "",
					"caddy.gzip":              "",
					"caddy.basicauth":         "/ user password",
					"caddy.tls.dns":           "route53",
					"caddy.rewrite_0":         "/path1 /path2",
					"caddy.rewrite_1":         "/path3 /path4",
					"caddy.limits.header":     "100kb",
					"caddy.limits.body_0":     "/path1 2mb",
					"caddy.limits.body_1":     "/path2 4mb",
				},
			},
		},
	}

	const expected string = "service.testdomain.com {\n" +
		"  basicauth / user password\n" +
		"  gzip\n" +
		"  limits {\n" +
		"    body /path1 2mb\n" +
		"    body /path2 4mb\n" +
		"    header 100kb\n" +
		"  }\n" +
		"  proxy / service:5000/api {\n" +
		"    healthcheck /health\n" +
		"    transparent\n" +
		"    websocket\n" +
		"  }\n" +
		"  rewrite /path1 /path2\n" +
		"  rewrite /path3 /path4\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n"

	testSingleService(t, service, expected)
}

func TestAddServiceMacroLabels(t *testing.T) {
	var service = swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "service",
				Labels: map[string]string{
					"caddy.address":           "service.testdomain.com",
					"caddy.targetport":        "5000",
					"caddy.targetpath":        "/api",
					"caddy.proxy.healthcheck": "/health",
					"caddy.proxy.websocket":   "",
					"caddy.basicauth":         "/ user password",
					"caddy.tls.dns":           "route53",
				},
			},
		},
	}

	const expected string = "service.testdomain.com {\n" +
		"  basicauth / user password\n" +
		"  gzip\n" +
		"  proxy / service:5000/api {\n" +
		"    healthcheck /health\n" +
		"    transparent\n" +
		"    websocket\n" +
		"  }\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n"

	testSingleService(t, service, expected)
}

func testSingleService(t *testing.T, service swarm.Service, expected string) {
	var buffer bytes.Buffer
	addServiceToCaddyFile(&buffer, &service)
	var content = buffer.String()
	assert.Equal(t, expected, content)
}
