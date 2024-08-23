package generator

import (
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/lucaslorentz/caddy-docker-proxy/v2/caddyfile"
)

type targetsProvider func() ([]string, error)
type domainProvider func() (string, error)

func labelsToCaddyfile(labels map[string]string, templateData interface{}, getDomain domainProvider, getTargets targetsProvider) (*caddyfile.Container, error) {
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
		"h2c": func() string {
			return "h2c"
		},
		"addDomain": func(host string) (string, error) {
			localDomain, err := getDomain()
			if err == nil {
				host = fmt.Sprintf("%[1]s %[1]s.%[2]s", host, localDomain)
			}
			return host, err
		},
	}

	return caddyfile.FromLabels(labels, templateData, funcMap)
}
