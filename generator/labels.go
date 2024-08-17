package generator

import (
	"strconv"
	"strings"
	"text/template"

	"github.com/lucaslorentz/caddy-docker-proxy/v2/caddyfile"
)

type targetsProvider func() ([]string, error)
type localDomainProvider func() (string, error)

func labelsToCaddyfile(labels map[string]string, templateData interface{}, getTargets targetsProvider, getLocalDomain localDomainProvider) (*caddyfile.Container, error) {
	funcMap := template.FuncMap{
		"domain": func(options ...interface{}) (string, error) {
			localDomain, err := getLocalDomain()
			transformed := []string{}
			for _, param := range options {
				if host, isHost := param.(string); isHost {
					transformed = append(transformed, host, host+"."+localDomain)
				}
			}
			return strings.Join(transformed, " "), err
		},
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
		"h2c": func() string {
			return "h2c"
		},
	}

	return caddyfile.FromLabels(labels, templateData, funcMap)
}
