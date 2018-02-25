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

	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err == nil {
		for _, container := range containers {
			addContainerToCaddyFile(&buffer, &container)
		}
	} else {
		addError(&buffer, err)
	}

	services, err := dockerClient.ServiceList(context.Background(), types.ServiceListOptions{})
	if err == nil {
		for _, service := range services {
			addServiceToCaddyFile(&buffer, &service)
		}
	} else {
		addError(&buffer, err)
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

func addContainerToCaddyFile(buffer *bytes.Buffer, container *types.Container) {
	ipAddress := getContainerIPAddress(container)
	var directives = parseDirectives(container.Labels, container, ipAddress)
	for _, name := range getSortedKeys(&directives.children) {
		writeDirective(buffer, directives.children[name], 0)
	}
}

func getContainerIPAddress(container *types.Container) string {
	for _, network := range container.NetworkSettings.Networks {
		return network.IPAddress
	}
	return ""
}

func addServiceToCaddyFile(buffer *bytes.Buffer, service *swarm.Service) {
	var directives = parseDirectives(service.Spec.Labels, service, service.Spec.Name)
	for _, name := range getSortedKeys(&directives.children) {
		writeDirective(buffer, directives.children[name], 0)
	}
}

func parseDirectives(labels map[string]string, templateData interface{}, proxyTarget string) *directiveData {
	rootDirective := &directiveData{}

	convertLabelsToDirectives(labels, templateData, rootDirective)

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
			proxyDirective.args = fmt.Sprintf("/ %s:%s", proxyTarget, targetPort.args)
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

func convertLabelsToDirectives(labels map[string]string, templateData interface{}, rootDirective *directiveData) {
	for label, value := range labels {
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
		directive.args = processVariables(templateData, value)
	}
}

func processVariables(data interface{}, content string) string {
	t, err := template.New("").Parse(content)
	if err != nil {
		log.Println(err)
		return content
	}
	var writer bytes.Buffer
	t.Execute(&writer, data)
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
