package plugin

import (
	"testing"

	_ "github.com/caddyserver/caddy/caddyhttp" // plug in the HTTP server type
	"github.com/stretchr/testify/assert"
)

func TestProcessCaddyfile(t *testing.T) {
	input := "(mysnippet) {\n" +
		"  gzip\n" +
		"}\n" +
		"service1.example.com {\n" +
		"  proxy / service1:5000/api {\n" +
		"    transparentt\n" +
		"  }\n" +
		"}\n" +
		"service2.example.com {\n" +
		"  status 200 /\n" +
		"  #Coment\n" +
		"  proxy / service2:5000/api {\n" +
		"    except /a /b\n" +
		"    websocket\n" +
		"    transparent\n" +
		"  }\n" +
		"  import mysnippet\n" +
		"}\n" +
		"service3.example.com {\n" +
		"  status 404 /\n" +
		"  basicauth /secret user \" a \\\" b\"\n" +
		"}\n"

	expected := "service2.example.com {\n" +
		"  gzip\n" +
		"  proxy / service2:5000/api {\n" +
		"    except /a /b\n" +
		"    websocket\n" +
		"    transparent\n" +
		"  }\n" +
		"  status 200 /\n" +
		"}\n" +
		"service3.example.com {\n" +
		"  basicauth /secret user \" a \\\" b\"\n" +
		"  status 404 /\n" +
		"}\n"

	result := string(ProcessCaddyfile([]byte(input)))

	assert.Equal(t, expected, result)
}
