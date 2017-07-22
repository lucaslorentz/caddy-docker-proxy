package plugin

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

var dockerClient *client.Client

// GenerateCaddyFile generates a caddy file config from docker swarm
func GenerateCaddyFile() []byte {
	var buffer bytes.Buffer

	if dockerClient == nil {
		var err error
		if dockerClient, err = client.NewEnvClient(); err != nil {
			addError(&buffer, err)
			return buffer.Bytes()
		}
	}

	services, err := dockerClient.ServiceList(context.Background(), types.ServiceListOptions{})
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
	var rootDirective = parseDirectives(service)
	for _, directive := range rootDirective.children {
		writeDirective(buffer, directive, 0)
	}
}

func parseDirectives(service *swarm.Service) *Directive {
	rootDirective := &Directive{}

	address := service.Spec.Labels["caddy.address"]
	targetPort := service.Spec.Labels["caddy.targetport"]

	if address != "" && targetPort != "" {
		targetPath := service.Spec.Labels["caddy.targetpath"]

		proxyDirective := &Directive{
			name: "proxy",
			args: fmt.Sprintf("/ %s:%s%s", service.Spec.Name, targetPort, targetPath),
			children: map[string]*Directive{
				"transparent": &Directive{
					name: "transparent",
				},
			},
		}

		siteDirective := &Directive{
			name: address,
			children: map[string]*Directive{
				"proxy": proxyDirective,
				"gzip": &Directive{
					name: "gzip",
				},
			},
		}

		rootDirective.children = map[string]*Directive{
			"caddy": siteDirective,
		}

		delete(service.Spec.Labels, "caddy.address")
		delete(service.Spec.Labels, "caddy.targetport")
		delete(service.Spec.Labels, "caddy.targetpath")
	}

	parseAutomappedDirectives(service, rootDirective)

	return rootDirective
}

func getOrCreateDirective(directive *Directive, path string) {
	for i, p := range strings.Split(path, ".") {
		if d, ok := directive.children[p]; ok {
			directive = d
		} else {
			if directive.children == nil {
				directive.children = map[string]*Directive{}
			}
			var newDirective = Directive{}
			if i > 0 {
				newDirective.name = removeSuffix(p)
			}
			directive.children[p] = &newDirective
			directive = &newDirective
		}
	}
}

func parseAutomappedDirectives(service *swarm.Service, rootDirective *Directive) {
	for label, value := range service.Spec.Labels {
		if !strings.HasPrefix(label, "caddy") {
			continue
		}
		directive := rootDirective
		path := strings.Split(label, ".")
		for i, p := range path {
			if d, ok := directive.children[p]; ok {
				directive = d
			} else {
				if directive.children == nil {
					directive.children = map[string]*Directive{}
				}
				var newDirective = Directive{}
				if i > 0 {
					newDirective.name = removeSuffix(p)
				}
				directive.children[p] = &newDirective
				directive = &newDirective
			}
		}
		directive.args = processVariables(service, value)
	}
}

func processVariables(service *swarm.Service, content string) string {
	t, err := template.New("").Parse(content)
	if err != nil {
		log.Println(err)
		return content
	}
	var writer bytes.Buffer
	t.Execute(&writer, service)
	return writer.String()
}

func writeDirective(buffer *bytes.Buffer, directive *Directive, level int) {
	buffer.WriteString(strings.Repeat(" ", level*2))
	if directive.name != "" {
		buffer.WriteString(directive.name)
	}
	if directive.name != "" && directive.args != "" {
		buffer.WriteString(" ")
	}
	if directive.args != "" {
		buffer.WriteString(directive.args)
	}
	if directive.children != nil {
		buffer.WriteString(" {\n")
		for _, name := range getSortedKeys(&directive.children) {
			subdirective := directive.children[name]
			writeDirective(buffer, subdirective, level+1)
		}
		buffer.WriteString(strings.Repeat(" ", level*2) + "}")
	}
	buffer.WriteString("\n")
}

func removeSuffix(name string) string {
	return strings.Split(name, "_")[0]
}

func getSortedKeys(m *map[string]*Directive) []string {
	var keys = getKeys(m)
	sort.Strings(keys)
	return keys
}

func getKeys(m *map[string]*Directive) []string {
	var keys []string
	for k := range *m {
		keys = append(keys, k)
	}
	return keys
}

type Directive struct {
	name     string
	args     string
	children map[string]*Directive
}
