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
	"math/rand"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

var swarmAvailabilityCacheInterval = 1 * time.Minute

var defaultLabelPrefix = "caddy"

// CaddyfileGenerator generates caddyfile
type CaddyfileGenerator struct {
	labelRegex           *regexp.Regexp
	proxyServiceTasks    bool
	dockerClient         *client.Client
	caddyNetworks        map[string]bool
	swarmIsAvailable     bool
	swarmIsAvailableTime time.Time
}

var isTrue = regexp.MustCompile("(?i)^(true|yes|1)$")
var suffixRegex = regexp.MustCompile("_\\d+$")

var labelPrefixFlag string
var proxyServiceTasksFlag bool

func init() {
	flag.StringVar(&labelPrefixFlag, "docker-label-prefix", defaultLabelPrefix, "Prefix for Docker labels")
	flag.BoolVar(&proxyServiceTasksFlag, "proxy-service-tasks", false, "Proxy to service tasks instead of VIP")
}

// GeneratorOptions are the options for generator
type GeneratorOptions struct {
	labelPrefix       string
	proxyServiceTasks bool
}

// GetGeneratorOptions creates generator options from cli flags and environment variables
func GetGeneratorOptions() *GeneratorOptions {
	options := GeneratorOptions{}

	if labelPrefixEnv := os.Getenv("CADDY_DOCKER_LABEL_PREFIX"); labelPrefixEnv != "" {
		options.labelPrefix = labelPrefixEnv
	} else {
		options.labelPrefix = labelPrefixFlag
	}

	if proxyServiceTasksEnv := os.Getenv("CADDY_DOCKER_PROXY_SERVICE_TASKS"); proxyServiceTasksEnv != "" {
		options.proxyServiceTasks = isTrue.MatchString(proxyServiceTasksEnv)
	} else {
		options.proxyServiceTasks = proxyServiceTasksFlag
	}

	return &options
}

// CreateGenerator creates a new generator
func CreateGenerator(dockerClient *client.Client, options *GeneratorOptions) *CaddyfileGenerator {
	generator := CaddyfileGenerator{}

	generator.dockerClient = dockerClient

	var labelRegexString = fmt.Sprintf("^%s(_\\d+)?(\\.|$)", options.labelPrefix)
	generator.labelRegex = regexp.MustCompile(labelRegexString)

	generator.proxyServiceTasks = options.proxyServiceTasks

	return &generator
}

// GenerateCaddyFile generates a caddy file config from docker swarm
func (g *CaddyfileGenerator) GenerateCaddyFile() []byte {
	var buffer bytes.Buffer

	if g.caddyNetworks == nil {
		networks, err := g.getCaddyNetworks()
		if err == nil {
			g.caddyNetworks = map[string]bool{}
			for _, network := range networks {
				g.caddyNetworks[network] = true
			}
		} else {
			g.addComment(&buffer, err.Error())
		}
	}

	if time.Since(g.swarmIsAvailableTime) > swarmAvailabilityCacheInterval {
		g.checkSwarmAvailability(time.Time.IsZero(g.swarmIsAvailableTime))
		g.swarmIsAvailableTime = time.Now()
	}

	directives := make(map[string][]byte)

	containers, err := g.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err == nil {
		for _, container := range containers {
			dContent := g.addContainerToCaddyFile(&container)
			for _, d := range dContent {
				if d.name != "" {
					directives[d.name] = d.content.Bytes()
				}
			}
		}
	} else {
		g.addComment(&buffer, err.Error())
	}

	if g.swarmIsAvailable {
		services, err := g.dockerClient.ServiceList(context.Background(), types.ServiceListOptions{})
		if err == nil {
			for _, service := range services {
				dContent := g.addServiceToCaddyFile(&service)
				for _, d := range dContent {
					if d.name != "" {
						directives[d.name] = d.content.Bytes()
					}
				}
			}
		} else {
			g.addComment(&buffer, err.Error())
		}
	} else {
		g.addComment(&buffer, "Skipping services because swarm is not available")
	}

	var d_keys []string
	for key, _ := range directives {
		d_keys = append(d_keys, key)
	}

	sort.Strings(d_keys)

	for _, k := range d_keys {
		buffer.Write(directives[k])
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

func (g *CaddyfileGenerator) checkSwarmAvailability(isFirstCheck bool) {
	info, err := g.dockerClient.Info(context.Background())
	if err == nil {
		newSwarmIsAvailable := info.Swarm.LocalNodeState == swarm.LocalNodeStateActive
		if isFirstCheck || newSwarmIsAvailable != g.swarmIsAvailable {
			log.Printf("[INFO] Swarm is available: %v\n", newSwarmIsAvailable)
		}
		g.swarmIsAvailable = newSwarmIsAvailable
	} else {
		log.Printf("[ERROR] Swarm availability check failed: %v\n", err.Error())
		g.swarmIsAvailable = false
	}
}

func (g *CaddyfileGenerator) getCaddyNetworks() ([]string, error) {
	containerID, err := getCaddyContainerID()
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Caddy ContainerID: %v\n", containerID)
	container, err := g.dockerClient.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return nil, err
	}

	var networks []string
	for _, network := range container.NetworkSettings.Networks {
		networkInfo, err := g.dockerClient.NetworkInspect(context.Background(), network.NetworkID, types.NetworkInspectOptions{})
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

func (g *CaddyfileGenerator) addComment(buffer *bytes.Buffer, text string) {
	for _, line := range strings.Split(text, `\n`) {
		buffer.WriteString(fmt.Sprintf("# %s\n", line))
	}
}

func (g *CaddyfileGenerator) addContainerToCaddyFile(container *types.Container) (dContent []directiveContent) {
	directiveMap, err := g.parseDirectives(container.Labels, container, func() (string, error) {
		return g.getContainerIPAddress(container)
	})
	if err != nil {
		var d directiveContent
		d.name = fmt.Sprintf("%d", rand.Int())
		g.addComment(&d.content, err.Error())
		dContent = append(dContent, d)
		return
	}
	for _, name := range getSortedKeys(directiveMap) {
		var d directiveContent
		d.name = directiveMap[name].name
		writeDirective(&d.content, directiveMap[name], 0)
		dContent = append(dContent, d)
	}

	return
}

func (g *CaddyfileGenerator) getContainerIPAddress(container *types.Container) (string, error) {
	for _, network := range container.NetworkSettings.Networks {
		if _, isCaddyNetwork := g.caddyNetworks[network.NetworkID]; isCaddyNetwork {
			return network.IPAddress, nil
		}
	}
	return "", fmt.Errorf("Container %v and caddy are not in same network", container.ID)
}

func (g *CaddyfileGenerator) addServiceToCaddyFile(service *swarm.Service) (dContent []directiveContent) {
	directiveMap, err := g.parseDirectives(service.Spec.Labels, service, func() (string, error) {
		return g.getServiceProxyTarget(service)
	})
	if err != nil {
		var d directiveContent
		d.name = fmt.Sprintf("%d", rand.Int())
		g.addComment(&d.content, err.Error())
		dContent = append(dContent, d)
		return
	}
	for _, name := range getSortedKeys(directiveMap) {
		var d directiveContent
		d.name = directiveMap[name].name
		writeDirective(&d.content, directiveMap[name], 0)
		dContent = append(dContent, d)
	}

	return
}

func (g *CaddyfileGenerator) getServiceProxyTarget(service *swarm.Service) (string, error) {
	_, err := g.getServiceIPAddress(service)
	if err != nil {
		return "", err
	}

	if g.proxyServiceTasks {
		return "tasks." + service.Spec.Name, nil
	}

	return service.Spec.Name, nil
}

func (g *CaddyfileGenerator) getServiceIPAddress(service *swarm.Service) (string, error) {
	for _, virtualIP := range service.Endpoint.VirtualIPs {
		if _, isCaddyNetwork := g.caddyNetworks[virtualIP.NetworkID]; isCaddyNetwork {
			return virtualIP.Addr, nil
		}
	}

	return "", fmt.Errorf("Service %v and caddy are not in same network", service.ID)
}

func (g *CaddyfileGenerator) parseDirectives(labels map[string]string, templateData interface{}, getProxyTarget func() (string, error)) (map[string]*directiveData, error) {
	originalMap := g.convertLabelsToDirectives(labels, templateData)

	convertedMap := map[string]*directiveData{}

	//Convert basic labels
	for _, directive := range originalMap {
		address := directive.children["address"]

		if address != nil {
			directive.name = address.args

			targetPort := directive.children["targetport"]
			targetPath := directive.children["targetpath"]
			targetProtocol := directive.children["targetprotocol"]

			proxyDirective := getOrCreateDirective(directive.children, "proxy", false)
			proxyTarget, err := getProxyTarget()
			if err != nil {
				return nil, err
			}

			proxyDirective.args = "/ "

			if targetProtocol != nil {
				proxyDirective.args += targetProtocol.args + "://"
			}

			proxyDirective.args += proxyTarget

			if targetPort != nil {
				proxyDirective.args += ":" + targetPort.args
			}

			if targetPath != nil {
				proxyDirective.args += targetPath.args
			}
		}

		delete(directive.children, "address")
		delete(directive.children, "targetport")
		delete(directive.children, "targetpath")
		delete(directive.children, "targetprotocol")

		convertedMap[directive.name] = directive
	}

	return convertedMap, nil
}

func getOrCreateDirective(directiveMap map[string]*directiveData, path string, skipFirstDirectiveName bool) (directive *directiveData) {
	currentMap := directiveMap
	for i, p := range strings.Split(path, ".") {
		if d, ok := currentMap[p]; ok {
			directive = d
			currentMap = d.children
		} else {
			directive = &directiveData{
				children: map[string]*directiveData{},
			}
			if !skipFirstDirectiveName || i > 0 {
				directive.name = removeSuffix(p)
			}
			currentMap[p] = directive
			currentMap = directive.children
		}
	}
	return
}

func (g *CaddyfileGenerator) convertLabelsToDirectives(labels map[string]string, templateData interface{}) map[string]*directiveData {
	directiveMap := map[string]*directiveData{}

	for label, value := range labels {
		if !g.labelRegex.MatchString(label) {
			continue
		}
		directive := getOrCreateDirective(directiveMap, label, true)
		directive.args = processVariables(templateData, value)
	}

	return directiveMap
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
	if len(directive.children) > 0 {
		buffer.WriteString(" {\n")
		for _, name := range getSortedKeys(directive.children) {
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

func getSortedKeys(m map[string]*directiveData) []string {
	var keys = getKeys(m)
	sort.Strings(keys)
	return keys
}

func getKeys(m map[string]*directiveData) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

type directiveData struct {
	name     string
	args     string
	children map[string]*directiveData
}

type directiveContent struct {
	name    string
	content bytes.Buffer
}
