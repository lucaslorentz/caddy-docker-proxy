package generator

import (
	"errors"
	"sort"
	"slices"
	"strconv"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/caddyfile"
)

type targetsProvider func() ([]string, error)
type exposedPortsProvider func() ([]int, error)

func labelsToCaddyfile(labels map[string]string, templateData interface{}, getTargets targetsProvider, getExposedPorts exposedPortsProvider) (*caddyfile.Container, error) {
	funcMap := sprig.TxtFuncMap()
	funcMap["upstreams"] = func(options ...interface{}) (string, error) {
		targets, err := getTargets()
		sort.Strings(targets)
		transformed := []string{}
		for _, target := range targets {
			for _, param := range options {
				if protocol, isProtocol := param.(string); isProtocol {
					target = protocol + "://" + target
				} else if port, isPort := param.(int); isPort {
					if port == 0 {
						ports, portsErr := getExposedPorts()
						if portsErr != nil {
							err = errors.Join(err, portsErr)
						} else {
							port = slices.Min(ports)
						}
					}
					target = target + ":" + strconv.Itoa(port)
				}
			}
			transformed = append(transformed, target)
		}
		sort.Strings(transformed)
		return strings.Join(transformed, " "), err
	}
	funcMap["http"] = func() string { return "http" }
	funcMap["https"] = func() string { return "https" }
	funcMap["h2c"] = func() string { return "h2c" }

	return caddyfile.FromLabels(labels, templateData, funcMap)
}
