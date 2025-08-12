package generator

import (
	"strconv"
	"strings"
	"text/template"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/lucaslorentz/caddy-docker-proxy/v2/caddyfile"
)

type targetsProvider func() ([]string, error)

func labelsToCaddyfile(labels map[string]string, templateData interface{}, getTargets targetsProvider) (*caddyfile.Container, error) {
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
		
		// Environment Variable Functions
		"env": func(key string) string {
			if _, ok := templateData.(*types.Container); ok {
				// For containers, environment variables are in Config.Env
				// Container inspection would be needed to get full Config data
				// The types.Container only contains runtime info, not creation config
				// TODO: Implement container inspection to get Config.Env
				return ""
			} else if service, ok := templateData.(*swarm.Service); ok {
				// For services, check TaskTemplate.ContainerSpec.Env
				if service.Spec.TaskTemplate.ContainerSpec != nil {
					for _, env := range service.Spec.TaskTemplate.ContainerSpec.Env {
						parts := strings.SplitN(env, "=", 2)
						if len(parts) == 2 && parts[0] == key {
							return parts[1]
						}
					}
				}
			}
			return ""
		},
		
		"hasEnv": func(key string) bool {
			if _, ok := templateData.(*types.Container); ok {
				// For containers - need inspection
				return false
			} else if service, ok := templateData.(*swarm.Service); ok {
				if service.Spec.TaskTemplate.ContainerSpec != nil {
					for _, env := range service.Spec.TaskTemplate.ContainerSpec.Env {
						parts := strings.SplitN(env, "=", 2)
						if len(parts) >= 1 && parts[0] == key {
							return true
						}
					}
				}
			}
			return false
		},
		
		// Container Metadata Functions
		"containerName": func() string {
			if container, ok := templateData.(*types.Container); ok {
				if len(container.Names) > 0 {
					// Remove the leading "/" from container name
					return strings.TrimPrefix(container.Names[0], "/")
				}
			} else if service, ok := templateData.(*swarm.Service); ok {
				return service.Spec.Name
			}
			return ""
		},
		
		"imageName": func() string {
			if container, ok := templateData.(*types.Container); ok {
				return container.Image
			} else if service, ok := templateData.(*swarm.Service); ok {
				if service.Spec.TaskTemplate.ContainerSpec != nil {
					return service.Spec.TaskTemplate.ContainerSpec.Image
				}
			}
			return ""
		},
		
		"imageTag": func() string {
			imageName := ""
			if container, ok := templateData.(*types.Container); ok {
				imageName = container.Image
			} else if service, ok := templateData.(*swarm.Service); ok {
				if service.Spec.TaskTemplate.ContainerSpec != nil {
					imageName = service.Spec.TaskTemplate.ContainerSpec.Image
				}
			}
			
			// Extract tag from image name (e.g., "nginx:latest" -> "latest")
			parts := strings.Split(imageName, ":")
			if len(parts) > 1 {
				return parts[len(parts)-1]
			}
			return "latest" // Default tag
		},
		
		// Label Functions
		"label": func(key string) string {
			if container, ok := templateData.(*types.Container); ok {
				if val, exists := container.Labels[key]; exists {
					return val
				}
			} else if service, ok := templateData.(*swarm.Service); ok {
				if val, exists := service.Spec.Labels[key]; exists {
					return val
				}
			}
			return ""
		},
		
		"hasLabel": func(key string) bool {
			if container, ok := templateData.(*types.Container); ok {
				_, exists := container.Labels[key]
				return exists
			} else if service, ok := templateData.(*swarm.Service); ok {
				_, exists := service.Spec.Labels[key]
				return exists
			}
			return false
		},
		
		// Network Functions  
		"primaryIP": func() string {
			if container, ok := templateData.(*types.Container); ok {
				// Return the first available IP address
				for _, network := range container.NetworkSettings.Networks {
					if network.IPAddress != "" {
						return network.IPAddress
					}
				}
			}
			return ""
		},
		
		"networkIP": func(networkName string) string {
			if container, ok := templateData.(*types.Container); ok {
				if network, exists := container.NetworkSettings.Networks[networkName]; exists {
					return network.IPAddress
				}
			}
			return ""
		},

		"networks": func() []string {
			networks := []string{}
			if container, ok := templateData.(*types.Container); ok {
				for networkName := range container.NetworkSettings.Networks {
					networks = append(networks, networkName)
				}
			}
			return networks
		},

		// Volume and Mount Functions
		"mountSource": func(mountPoint string) string {
			if container, ok := templateData.(*types.Container); ok {
				for _, mount := range container.Mounts {
					if mount.Destination == mountPoint {
						return mount.Source
					}
				}
			} else if service, ok := templateData.(*swarm.Service); ok {
				if service.Spec.TaskTemplate.ContainerSpec != nil {
					for _, mount := range service.Spec.TaskTemplate.ContainerSpec.Mounts {
						if mount.Target == mountPoint {
							return mount.Source
						}
					}
				}
			}
			return ""
		},

		"bindMounts": func() []string {
			mounts := []string{}
			if container, ok := templateData.(*types.Container); ok {
				for _, mount := range container.Mounts {
					if mount.Type == "bind" {
						mounts = append(mounts, mount.Source)
					}
				}
			} else if service, ok := templateData.(*swarm.Service); ok {
				if service.Spec.TaskTemplate.ContainerSpec != nil {
					for _, mount := range service.Spec.TaskTemplate.ContainerSpec.Mounts {
						if mount.Type == "bind" {
							mounts = append(mounts, mount.Source)
						}
					}
				}
			}
			return mounts
		},

		"volumeMounts": func() []string {
			mounts := []string{}
			if container, ok := templateData.(*types.Container); ok {
				for _, mount := range container.Mounts {
					if mount.Type == "volume" {
						mounts = append(mounts, mount.Name)
					}
				}
			} else if service, ok := templateData.(*swarm.Service); ok {
				if service.Spec.TaskTemplate.ContainerSpec != nil {
					for _, mount := range service.Spec.TaskTemplate.ContainerSpec.Mounts {
						if mount.Type == "volume" {
							mounts = append(mounts, mount.Source)
						}
					}
				}
			}
			return mounts
		},

		"hasMount": func(mountPoint string) bool {
			if container, ok := templateData.(*types.Container); ok {
				for _, mount := range container.Mounts {
					if mount.Destination == mountPoint {
						return true
					}
				}
			} else if service, ok := templateData.(*swarm.Service); ok {
				if service.Spec.TaskTemplate.ContainerSpec != nil {
					for _, mount := range service.Spec.TaskTemplate.ContainerSpec.Mounts {
						if mount.Target == mountPoint {
							return true
						}
					}
				}
			}
			return false
		},

		// Port Mapping Functions
		"portMapping": func(containerPort int) string {
			if container, ok := templateData.(*types.Container); ok {
				for _, port := range container.Ports {
					if int(port.PrivatePort) == containerPort {
						if port.PublicPort > 0 {
							return strconv.Itoa(int(port.PublicPort))
						}
					}
				}
			}
			return ""
		},

		"exposedPorts": func() []string {
			ports := []string{}
			if container, ok := templateData.(*types.Container); ok {
				for _, port := range container.Ports {
					if port.PublicPort > 0 {
						ports = append(ports, strconv.Itoa(int(port.PublicPort)))
					}
				}
			}
			return ports
		},
		
		// Container State Functions (basic implementations)
		"isRunning": func() bool {
			if container, ok := templateData.(*types.Container); ok {
				return container.State == "running"
			}
			// For services, assume running if exists
			return true
		},
		
		"isHealthy": func() bool {
			if container, ok := templateData.(*types.Container); ok {
				// Basic health check - would need container inspection for detailed health
				return container.State == "running"
			}
			// For services, assume healthy if exists
			return true
		},
	}

	return caddyfile.FromLabels(labels, templateData, funcMap)
}
