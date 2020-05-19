package generator

import (
	"strconv"
	"strings"
	"text/template"

	"github.com/lucaslorentz/caddy-docker-proxy/plugin/v2/caddyfile"
)

type targetsProvider func() ([]string, error)

func labelsToCaddyfile(labels map[string]string, templateData interface{}, getTargets targetsProvider) (*caddyfile.Block, error) {
	funcMap := template.FuncMap{
		"upstreams": func(options ...interface{}) (string, error) {
			targets, err := getTargets()
			transformed := []string{}
			for _, target := range targets {
				for _, param := range options {
					if protocol, isProtocol := param.(string); isProtocol {
						target = protocol + "://" + target
					} else if port, isPort := param.(int); isPort {
						target = target + ":" + strconv.Itoa(port)
					}
				}
				transformed = append(transformed, target)
			}
			return strings.Join(transformed, " "), err
		},
		"http": func() string {
			return "http"
		},
		"https": func() string {
			return "https"
		},
	}

	return caddyfile.FromLabels(labels, templateData, funcMap)
}
