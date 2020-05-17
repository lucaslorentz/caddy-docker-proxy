package caddyfile

import (
	"testing"

	_ "github.com/caddyserver/caddy/v2/modules/standard" // plug standard HTTP modules

	"github.com/stretchr/testify/assert"
)

func TestProcess(t *testing.T) {
	input := "(mysnippet) {\n" +
		"	encode gzip\n" +
		"}\n" +
		"service1.example.com {\n" +
		"	reverse_proxy service1:5000/api {\n" +
		"		invalid\n" +
		"	}\n" +
		"}\n" +
		"service2.example.com {\n" +
		"	respond 200 /\n" +
		"	#Coment\n" +
		"	reverse_proxy service2:5000/api {\n" +
		"		health_path /health\n" +
		"	}\n" +
		"	import mysnippet\n" +
		"}\n" +
		"service3.example.com {\n" +
		"	respond 404 /\n" +
		"	basicauth /secret {\n" +
		"		user \" a \\ b\"\n" +
		"	}\n" +
		"}\n"

	expected := "service2.example.com {\n" +
		"	respond 200 /\n" +
		"	reverse_proxy service2:5000/api {\n" +
		"		health_path /health\n" +
		"	}\n" +
		"	encode gzip\n" +
		"}\n" +
		"service3.example.com {\n" +
		"	respond 404 /\n" +
		"	basicauth /secret {\n" +
		"		user \" a \\ b\"\n" +
		"	}\n" +
		"}\n"

	result := Process([]byte(input))

	assert.Equal(t, expected, string(result))
}
