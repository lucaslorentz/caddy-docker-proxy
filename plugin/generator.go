package plugin

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
)

var swarmAvailabilityCacheInterval = 1 * time.Minute

var defaultLabelPrefix = "caddy"

// CaddyfileGenerator generates caddyfile
type CaddyfileGenerator struct {
	labelPrefix          string
	labelRegex           *regexp.Regexp
	proxyServiceTasks    bool
	dockerClient         DockerClient
	dockerUtils          DockerUtils
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
func CreateGenerator(dockerClient DockerClient, dockerUtils DockerUtils, options *GeneratorOptions) *CaddyfileGenerator {
	var labelRegexString = fmt.Sprintf("^%s(_\\d+)?(\\.|$)", options.labelPrefix)

	return &CaddyfileGenerator{
		dockerClient:      dockerClient,
		dockerUtils:       dockerUtils,
		labelPrefix:       options.labelPrefix,
		labelRegex:        regexp.MustCompile(labelRegexString),
		proxyServiceTasks: options.proxyServiceTasks,
	}
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

	directives := map[string]*directiveData{}

	containers, err := g.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err == nil {
		for _, container := range containers {
			containerDirectives, err := g.getContainerDirectives(&container)
			if err == nil {
				for k, directive := range containerDirectives {
					directives[k] = mergeDirectives(directives[k], directive)
				}
			} else {
				g.addComment(&buffer, err.Error())
			}
		}
	} else {
		g.addComment(&buffer, err.Error())
	}

	if g.swarmIsAvailable {
		services, err := g.dockerClient.ServiceList(context.Background(), types.ServiceListOptions{})
		if err == nil {
			for _, service := range services {
				serviceDirectives, err := g.getServiceDirectives(&service)
				if err == nil {
					for k, directive := range serviceDirectives {
						directives[k] = mergeDirectives(directives[k], directive)
					}
				} else {
					g.addComment(&buffer, err.Error())
				}
			}
		} else {
			g.addComment(&buffer, err.Error())
		}
	} else {
		g.addComment(&buffer, "Skipping services because swarm is not available")
	}

	if g.swarmIsAvailable {
		configs, err := g.dockerClient.ConfigList(context.Background(), types.ConfigListOptions{})
		if err == nil {
			for _, config := range configs {
				if _, hasLabel := config.Spec.Labels[g.labelPrefix]; hasLabel {
					fullConfig, _, err := g.dockerClient.ConfigInspectWithRaw(context.Background(), config.ID)
					if err == nil {
						buffer.Write(fullConfig.Spec.Data)
						buffer.WriteRune('\n')
					} else {
						g.addComment(&buffer, err.Error())
					}
				}
			}
		} else {
			g.addComment(&buffer, err.Error())
		}
	} else {
		g.addComment(&buffer, "Skipping configs because swarm is not available")
	}

	writeDirectives(&buffer, directives, 0)

	return buffer.Bytes()
}

func mergeDirectives(directiveA *directiveData, directiveB *directiveData) *directiveData {
	if directiveA == nil {
		return directiveB
	}
	if directiveB == nil {
		return directiveA
	}

	for keyB, subDirectiveB := range directiveB.children {
		if subDirectiveB.name == "proxy" {
			proxyB := subDirectiveB
			if proxyA, exists := directiveA.children[keyB]; exists {
				if len(proxyA.args) > 0 && len(proxyB.args) > 0 && proxyA.args[0] == proxyB.args[0] {
					proxyA.addArgs(proxyB.args[1:]...)
					continue
				}
			}
		}

		directiveA.children[keyB] = subDirectiveB
	}

	return directiveA
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
	containerID, err := g.dockerUtils.GetCurrentContainerID()
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

func (g *CaddyfileGenerator) getContainerDirectives(container *types.Container) (map[string]*directiveData, error) {
	return g.parseDirectives(container.Labels, container, func() (string, error) {
		return g.getContainerIPAddress(container)
	})
}

func (g *CaddyfileGenerator) getContainerIPAddress(container *types.Container) (string, error) {
	for _, network := range container.NetworkSettings.Networks {
		if _, isCaddyNetwork := g.caddyNetworks[network.NetworkID]; isCaddyNetwork {
			return network.IPAddress, nil
		}
	}
	return "", fmt.Errorf("Container %v and caddy are not in same network", container.ID)
}

func (g *CaddyfileGenerator) getServiceDirectives(service *swarm.Service) (map[string]*directiveData, error) {
	return g.parseDirectives(service.Spec.Labels, service, func() (string, error) {
		return g.getServiceProxyTarget(service)
	})
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

		if address != nil && len(address.args) > 0 {
			directive.args = address.args

			targetPort := directive.children["targetport"]
			targetPath := directive.children["targetpath"]
			targetProtocol := directive.children["targetprotocol"]

			proxyDirective := getOrCreateDirective(directive.children, "proxy", false)

			if len(proxyDirective.args) == 0 {
				proxyTarget, err := getProxyTarget()
				if err != nil {
					return nil, err
				}

				proxyDirective.addArgs("/")

				targetArg := ""
				if targetProtocol != nil && len(targetProtocol.args) > 0 {
					targetArg += targetProtocol.args[0] + "://"
				}
				targetArg += proxyTarget
				if targetPort != nil && len(targetPort.args) > 0 {
					targetArg += ":" + targetPort.args[0]
				}
				if targetPath != nil && len(targetPath.args) > 0 {
					targetArg += targetPath.args[0]
				}

				proxyDirective.addArgs(targetArg)
			}
		}

		delete(directive.children, "address")
		delete(directive.children, "targetport")
		delete(directive.children, "targetpath")
		delete(directive.children, "targetprotocol")

		//Move sites directive to main
		directive.name = strings.Join(directive.args, " ")
		directive.args = []string{}

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
		argsText := processVariables(templateData, value)
		directive.args = parseArgs(argsText)
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

func parseArgs(text string) []string {
	args := regSplit(text, "\\s+")
	if len(args) == 1 && args[0] == "" {
		return []string{}
	}
	return args
}

func regSplit(text string, delimeter string) []string {
	reg := regexp.MustCompile(delimeter)
	indexes := reg.FindAllStringIndex(text, -1)
	laststart := 0
	result := make([]string, len(indexes)+1)
	for i, element := range indexes {
		result[i] = text[laststart:element[0]]
		laststart = element[1]
	}
	result[len(indexes)] = text[laststart:len(text)]
	return result
}

func writeDirectives(buffer *bytes.Buffer, directives map[string]*directiveData, level int) {
	for _, name := range getSortedKeys(directives) {
		subdirective := directives[name]
		writeDirective(buffer, subdirective, level)
	}
}

func writeDirective(buffer *bytes.Buffer, directive *directiveData, level int) {
	buffer.WriteString(strings.Repeat(" ", level*2))
	if directive.name != "" {
		buffer.WriteString(directive.name)
	}
	if directive.name != "" && len(directive.args) > 0 {
		buffer.WriteString(" ")
	}
	if len(directive.args) > 0 {
		for index, arg := range directive.args {
			if index > 0 {
				buffer.WriteString(" ")
			}
			buffer.WriteString(arg)
		}
	}
	if len(directive.children) > 0 {
		buffer.WriteString(" {\n")
		writeDirectives(buffer, directive.children, level+1)
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
	args     []string
	children map[string]*directiveData
}

func (directive *directiveData) addArgs(args ...string) {
	directive.args = append(directive.args, args...)
}
