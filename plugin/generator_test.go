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

func TestAddServiceBasicLabels(t *testing.T) {
	var service = swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "service",
				Labels: map[string]string{
					"caddy.address":           "service.testdomain.com",
					"caddy.targetport":        "5000",
					"caddy.targetpath":        "/api",
					"caddy.proxy.healthcheck": "/health",
					"caddy.proxy.transparent": "",
					"caddy.proxy.websocket":   "",
					"caddy.basicauth":         "/ user password",
					"caddy.tls.dns":           "route53",
				},
			},
		},
	}

	const expected string = "service.testdomain.com {\n" +
		"  basicauth / user password\n" +
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

func TestAddServiceBasicLabelsMultipleConfigs(t *testing.T) {
	var service = swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "service",
				Labels: map[string]string{
					"caddy_0.address":           "service1.testdomain.com",
					"caddy_0.targetport":        "5000",
					"caddy_0.targetpath":        "/api",
					"caddy_0.proxy.healthcheck": "/health",
					"caddy_0.proxy.transparent": "",
					"caddy_0.proxy.websocket":   "",
					"caddy_0.basicauth":         "/ user password",
					"caddy_0.tls.dns":           "route53",
					"caddy_1.address":           "service2.testdomain.com",
					"caddy_1.targetport":        "5001",
					"caddy_1.tls.dns":           "route53",
				},
			},
		},
	}

	const expected string = "service1.testdomain.com {\n" +
		"  basicauth / user password\n" +
		"  proxy / service:5000/api {\n" +
		"    healthcheck /health\n" +
		"    transparent\n" +
		"    websocket\n" +
		"  }\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n" +
		"service2.testdomain.com {\n" +
		"  proxy / service:5001\n" +
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
