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
	var directives = parseDirectives(service)
	for _, name := range getSortedKeys(&directives.children) {
		writeDirective(buffer, directives.children[name], 0)
	}
}

func parseDirectives(service *swarm.Service) *directiveData {
	rootDirective := &directiveData{}

	convertLabelsToDirectives(service, rootDirective)

	//Convert basic labels
	for _, directive := range rootDirective.children {
		address := directive.children["address"]
		if address != nil {
			directive.name = address.args
		}

		targetPort := directive.children["targetport"]
		targetPath := directive.children["targetpath"]
		if targetPort != nil {
			proxyDirective := getOrCreateDirective(directive, "proxy")
			proxyDirective.args = fmt.Sprintf("/ %s:%s", service.Spec.Name, targetPort.args)
			if targetPath != nil {
				proxyDirective.args += targetPath.args
			}
		}

		delete(directive.children, "address")
		delete(directive.children, "targetport")
		delete(directive.children, "targetpath")
	}

	return rootDirective
}

func getOrCreateDirective(directive *directiveData, path string) *directiveData {
	currentDirective := directive

	for _, p := range strings.Split(path, ".") {
		if d, ok := currentDirective.children[p]; ok {
			currentDirective = d
		} else {
			if currentDirective.children == nil {
				currentDirective.children = map[string]*directiveData{}
			}
			var newDirective = directiveData{}
			newDirective.name = removeSuffix(p)
			currentDirective.children[p] = &newDirective
			currentDirective = &newDirective
		}
	}

	return currentDirective
}

func convertLabelsToDirectives(service *swarm.Service, rootDirective *directiveData) {
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
					directive.children = map[string]*directiveData{}
				}
				var newDirective = directiveData{}
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

func writeDirective(buffer *bytes.Buffer, directive *directiveData, level int) {
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

func getSortedKeys(m *map[string]*directiveData) []string {
	var keys = getKeys(m)
	sort.Strings(keys)
	return keys
}

func getKeys(m *map[string]*directiveData) []string {
	var keys []string
	for k := range *m {
		keys = append(keys, k)
	}
	return keys
}

type directiveData struct {
	name     string
	args     string
	children map[string]*directiveData
}
