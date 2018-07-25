package plugin

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

var proxyServiceTasks bool
var caddyNetworks map[string]bool

func init() {
	flag.BoolVar(&proxyServiceTasks, "proxy-service-tasks", false, "Proxy to service tasks instead of VIP")
}


func getCaddyLabelPrefix() string {
	if val := os.Getenv("CADDY_DOCKER_LABEL_PREFIX"); val != "" {
		return val
	}

	return "caddy"
}

var caddyLabelPrefix = getCaddyLabelPrefix()
var caddyLabelRegexString = fmt.Sprintf("^%s(_\\d+)?(\\.|$)", caddyLabelPrefix)
var caddyLabelRegex = regexp.MustCompile(caddyLabelRegexString)
var suffixRegex = regexp.MustCompile("_\\d+$")

// GenerateCaddyFile generates a caddy file config from docker swarm
func GenerateCaddyFile(dockerClient *client.Client) []byte {
	var buffer bytes.Buffer

	if caddyNetworks == nil {
		networks, err := getCaddyNetworks(dockerClient)
		if err == nil {
			caddyNetworks = map[string]bool{}
			for _, network := range networks {
				caddyNetworks[network] = true
			}
		} else {
			addComment(&buffer, err.Error())
		}
	}

	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err == nil {
		for _, container := range containers {
			addContainerToCaddyFile(&buffer, &container)
		}
	} else {
		addComment(&buffer, err.Error())
	}

	services, err := dockerClient.ServiceList(context.Background(), types.ServiceListOptions{})
	if err == nil {
		for _, service := range services {
			addServiceToCaddyFile(&buffer, &service)
		}
	} else {
		addComment(&buffer, err.Error())
	}

	if buffer.Len() == 0 {
		buffer.WriteString("# Empty file")
	}

	return buffer.Bytes()
}

func getCaddyContainerID() (string, error) {
	bytes, err := ioutil.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	if len(bytes) > 0 {
		cgroups := string(bytes)
		idRegex := regexp.MustCompile("docker/([A-Za-z0-9]+)")
		matches := idRegex.FindStringSubmatch(cgroups)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}
	return "", errors.New("Cannot find container id")
}

func getCaddyNetworks(dockerClient *client.Client) ([]string, error) {
	containerID, err := getCaddyContainerID()
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Caddy ContainerID: %v\n", containerID)
	container, err := dockerClient.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return nil, err
	}

	var networks []string
	for _, network := range container.NetworkSettings.Networks {
		networkInfo, err := dockerClient.NetworkInspect(context.Background(), network.NetworkID, types.NetworkInspectOptions{})
		if err != nil {
			return nil, err
		}
		if !networkInfo.Ingress {
			networks = append(networks, network.NetworkID)
		}
	}
	log.Printf("[INFO] Caddy Networks: %v\n", networks)

	return networks, nil
}

func addComment(buffer *bytes.Buffer, text string) {
	for _, line := range strings.Split(text, `\n`) {
		buffer.WriteString(fmt.Sprintf("# %s\n", line))
	}
}

func addContainerToCaddyFile(buffer *bytes.Buffer, container *types.Container) {
	directives, err := parseDirectives(container.Labels, container, func() (string, error) {
		return getContainerIPAddress(container)
	})
	if err != nil {
		addComment(buffer, err.Error())
		return
	}
	for _, name := range getSortedKeys(&directives.children) {
		writeDirective(buffer, directives.children[name], 0)
	}
}

func getContainerIPAddress(container *types.Container) (string, error) {
	for _, network := range container.NetworkSettings.Networks {
		if _, isCaddyNetwork := caddyNetworks[network.NetworkID]; isCaddyNetwork {
			return network.IPAddress, nil
		}
	}
	return "", fmt.Errorf("Container %v and caddy are not in same network", container.ID)
}

func addServiceToCaddyFile(buffer *bytes.Buffer, service *swarm.Service) {
	directives, err := parseDirectives(service.Spec.Labels, service, func() (string, error) {
		return getServiceProxyTarget(service)
	})
	if err != nil {
		addComment(buffer, err.Error())
		return
	}
	for _, name := range getSortedKeys(&directives.children) {
		writeDirective(buffer, directives.children[name], 0)
	}
}

func getServiceProxyTarget(service *swarm.Service) (string, error) {
	_, err := getServiceIPAddress(service)
	if err != nil {
		return "", err
	}

	if proxyServiceTasks {
		return "tasks." + service.Spec.Name, nil
	}

	return service.Spec.Name, nil
}

func getServiceIPAddress(service *swarm.Service) (string, error) {
	for _, virtualIP := range service.Endpoint.VirtualIPs {
		if _, isCaddyNetwork := caddyNetworks[virtualIP.NetworkID]; isCaddyNetwork {
			return virtualIP.Addr, nil
		}
	}
	return "", fmt.Errorf("Service %v and caddy are not in same network", service.ID)
}

func parseDirectives(labels map[string]string, templateData interface{}, getProxyTarget func() (string, error)) (*directiveData, error) {
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
		targetProtocol := directive.children["targetprotocol"]
		if targetPort != nil || targetProtocol != nil {
			proxyDirective := getOrCreateDirective(directive, "proxy")
			proxyTarget, err := getProxyTarget()
			if err != nil {
				return nil, err
			}

			proxyDirective.args = "/ "

			if targetProtocol != nil {
				proxyDirective.args += targetProtocol.args + "://"
			}

			proxyDirective.args += fmt.Sprintf("%s:%s", proxyTarget, targetPort.args)

			if targetPath != nil {
				proxyDirective.args += targetPath.args
			}
		}

		delete(directive.children, "address")
		delete(directive.children, "targetport")
		delete(directive.children, "targetpath")
		delete(directive.children, "targetprotocol")
	}

	return rootDirective, nil
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
		if !caddyLabelRegex.MatchString(label) {
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
	return suffixRegex.ReplaceAllString(name, "")
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
