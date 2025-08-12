# Enhanced Template Functions for Caddy-Docker-Proxy

This document describes the enhanced template function system that enables dynamic configuration of ALL Caddy directives using Docker container metadata.

## Overview

Template functions extract data from Docker containers and services to dynamically generate Caddyfile configurations. These functions unlock the full power of Caddy by making container metadata available for any directive, not just reverse proxying.

## Current Functions

### upstreams

Returns all addresses for the current Docker resource separated by whitespace.

**Usage:** `upstreams [protocol] [port]`

```yaml
labels:
  caddy.reverse_proxy: "{{upstreams}}"           # → 192.168.0.1 192.168.0.2
  caddy.reverse_proxy: "{{upstreams https}}"     # → https://192.168.0.1 https://192.168.0.2  
  caddy.reverse_proxy: "{{upstreams 8080}}"      # → 192.168.0.1:8080 192.168.0.2:8080
  caddy.reverse_proxy: "{{upstreams http 8080}}" # → http://192.168.0.1:8080 http://192.168.0.2:8080
```

### Protocol Helpers

Simple protocol string helpers:
- `http()` → "http"
- `https()` → "https" 
- `h2c()` → "h2c"

## Enhanced Functions (Planned)

### Volume and Storage Functions

Access Docker volume and mount information for static file serving.

#### bindMounts

Returns all bind mount source paths from the host.

**Usage:** `bindMounts`

```yaml
labels:
  caddy: static.example.com
  caddy.root: "* {{index (bindMounts) 0}}"  # Use first bind mount as root
  caddy.file_server: ""
```

#### mountSource

Returns the host source path for a specific container mount point.

**Usage:** `mountSource <container_path>`

```yaml
labels:
  caddy: docs.example.com
  caddy.root: "* {{mountSource "/app/docs"}}"  # → /host/path/to/docs
  caddy.file_server: ""
```

#### volumeMounts  

Returns all volume mount points in the container.

**Usage:** `volumeMounts`

```yaml
labels:
  caddy: files.example.com
  caddy.route.0_path: "{{range volumeMounts}}/{{.}}/* {{end}}"
  caddy.route.0_file_server: ""
```

#### hasMount

Checks if a specific path is mounted in the container.

**Usage:** `hasMount <container_path>`

```yaml
labels:
  caddy: app.example.com
  caddy.file_server: "{{if hasMount "/app/static"}}{{end}}"
  caddy.root: "{{if hasMount "/app/static"}}* {{mountSource "/app/static"}}{{end}}"
```

### Environment Variable Functions

Access container environment variables for dynamic configuration.

#### env

Returns the value of a specific environment variable.

**Usage:** `env <variable_name>`

```yaml
labels:
  caddy: secure.example.com
  caddy.basicauth: "/admin {{env "ADMIN_USER"}} {{env "ADMIN_PASS"}}"
  caddy.tls.email: "{{env "ACME_EMAIL"}}"
  caddy.reverse_proxy: "{{upstreams}}"
```

#### hasEnv

Checks if an environment variable exists.

**Usage:** `hasEnv <variable_name>`

```yaml
labels:
  caddy: app.example.com
  caddy.basicauth: "{{if hasEnv "ADMIN_USER"}}/admin {{env "ADMIN_USER"}} {{env "ADMIN_PASS"}}{{end}}"
  caddy.reverse_proxy: "{{upstreams}}"
```

#### envPrefix

Returns all environment variables with a specific prefix as a map.

**Usage:** `envPrefix <prefix>`

```yaml
labels:
  caddy: config.example.com
  caddy.templates: ""
  caddy.respond: |
    200 {
      body "Config: {{range $k, $v := envPrefix "APP_"}}{{$k}}={{$v}} {{end}}"
    }
```

### Network and Port Functions

Access container network configuration for advanced routing.

#### networkIP

Returns the container's IP address on a specific network.

**Usage:** `networkIP <network_name>`

```yaml
labels:
  caddy: multi.example.com
  caddy.route.0_path: "/api/*"
  caddy.route.0_reverse_proxy: "{{networkIP "backend"}}:8080"
  caddy.route.1_path: "/admin/*"
  caddy.route.1_reverse_proxy: "{{networkIP "management"}}:9090"
```

#### primaryIP

Returns the container's primary IP address.

**Usage:** `primaryIP`

```yaml
labels:
  caddy: direct.example.com
  caddy.reverse_proxy: "{{primaryIP}}:8080"
```

#### portMapping

Returns the host port mapped to a specific container port.

**Usage:** `portMapping <container_port>`

```yaml
labels:
  caddy: external.example.com
  caddy.reverse_proxy: "host.docker.internal:{{portMapping 8080}}"
```

#### exposedPorts

Returns all ports exposed by the container.

**Usage:** `exposedPorts`

```yaml
labels:
  caddy: multi-port.example.com
  caddy.respond: |
    200 {
      body "Available ports: {{range exposedPorts}}{{.}} {{end}}"
    }
```

#### networks

Returns all networks the container is connected to.

**Usage:** `networks`

```yaml
labels:
  caddy: status.example.com
  caddy.respond: |
    200 {
      body "Networks: {{range networks}}{{.}} {{end}}"
    }
```

### Container State Functions

Access container runtime state for conditional logic and health-based routing.

#### isRunning

Returns true if the container is currently running.

**Usage:** `isRunning`

```yaml
labels:
  caddy: app.example.com
  caddy.respond: "{{if not (isRunning)}}503 Service Temporarily Unavailable{{end}}"
  caddy.reverse_proxy: "{{if isRunning}}{{upstreams}}{{end}}"
```

#### isHealthy

Returns true if the container passes health checks.

**Usage:** `isHealthy`

```yaml
labels:
  caddy: health-aware.example.com
  caddy.route.0_@healthy: "path /*"
  caddy.route.0_@healthy.expression: "{{isHealthy}}"
  caddy.route.0_reverse_proxy: "@healthy {{upstreams}}"
  caddy.route.1_respond: "503 Service Unhealthy"
```

#### uptime

Returns the container's uptime as a duration.

**Usage:** `uptime`

```yaml
labels:
  caddy: status.example.com
  caddy.respond: |
    200 {
      body "Container uptime: {{uptime}}"
    }
```

#### restartCount

Returns the number of times the container has been restarted.

**Usage:** `restartCount`

```yaml
labels:
  caddy: monitoring.example.com
  caddy.respond: |
    200 {
      body "Restart count: {{restartCount}}"
    }
```

### Label and Metadata Functions

Access container labels and metadata for advanced configuration patterns.

#### label

Returns the value of a specific container label.

**Usage:** `label <label_key>`

```yaml
labels:
  caddy: dynamic.example.com
  caddy.rate_limit: "{{label "app.rate_limit"}} {{label "app.rate_window"}}"
  caddy.reverse_proxy: "{{upstreams}}"
  # Container also needs: app.rate_limit=100, app.rate_window=1m
```

#### hasLabel

Checks if a container has a specific label.

**Usage:** `hasLabel <label_key>`

```yaml
labels:
  caddy: conditional.example.com
  caddy.basicauth: "{{if hasLabel "app.auth_required"}}/admin user pass{{end}}"
  caddy.reverse_proxy: "{{upstreams}}"
```

#### labelPrefix

Returns all labels with a specific prefix as a map.

**Usage:** `labelPrefix <prefix>`

```yaml
labels:
  caddy: config.example.com
  caddy.header: "{{range $k, $v := labelPrefix "app.header."}}{{$k}} {{$v}}{{end}}"
  caddy.reverse_proxy: "{{upstreams}}"
  # Container also needs: app.header.X-Service=myapp, app.header.X-Version=1.0
```

#### containerName

Returns the container name without the leading slash.

**Usage:** `containerName`

```yaml
labels:
  caddy: "{{containerName}}.services.local"
  caddy.reverse_proxy: "{{upstreams}}"
  # If container name is "web-app", creates site: web-app.services.local
```

#### imageName

Returns the Docker image name used by the container.

**Usage:** `imageName`

```yaml
labels:
  caddy: app.example.com
  caddy.header.X-Image: "{{imageName}}"
  caddy.reverse_proxy: "{{upstreams}}"
```

#### imageTag

Returns just the tag portion of the Docker image.

**Usage:** `imageTag`

```yaml
labels:
  caddy: app.example.com
  caddy.header.X-Version: "{{imageTag}}"
  caddy.reverse_proxy: "{{upstreams}}"
```

## Real-World Usage Examples

### Static File Serving with Fallback API

```yaml
services:
  webapp:
    image: nginx:alpine
    volumes:
      - ./public:/usr/share/nginx/html:ro
    labels:
      caddy: myapp.example.com
      # Serve static files first, then proxy API requests
      caddy.route.0_@static.file: "{path}.html {path}/index.html"
      caddy.route.0_file_server: "@static {{mountSource "/usr/share/nginx/html"}}"
      caddy.route.1_@api.path_regexp: "^/api/"
      caddy.route.1_reverse_proxy: "@api {{upstreams}}"
```

### Environment-Based Authentication

```yaml
services:
  api:
    image: myapi:latest
    environment:
      - ADMIN_USER=admin
      - ADMIN_PASS=secret123
      - ENABLE_AUTH=true
    labels:
      caddy: api.example.com
      caddy.basicauth: "{{if env "ENABLE_AUTH"}}/admin {{env "ADMIN_USER"}} {{env "ADMIN_PASS"}}{{end}}"
      caddy.reverse_proxy: "{{upstreams}}"
```

### Health-Aware Load Balancing

```yaml
services:
  app:
    image: myapp:latest
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      retries: 3
    labels:
      caddy: app.example.com
      # Only proxy to healthy containers
      caddy.reverse_proxy: "{{if isHealthy}}{{upstreams}}{{else}}# Container unhealthy{{end}}"
      caddy.respond: "{{if not (isHealthy)}}503 Service Temporarily Unavailable{{end}}"
```

### Multi-Environment Configuration

```yaml
services:
  app:
    image: myapp:latest
    environment:
      - ENVIRONMENT=production
      - TLS_EMAIL=admin@example.com
    labels:
      caddy: myapp.example.com
      caddy.tls.email: "{{env "TLS_EMAIL"}}"
      caddy.header.X-Environment: "{{env "ENVIRONMENT"}}"
      caddy.reverse_proxy: "{{upstreams}}"
      # Add debug headers in non-production
      caddy.header.X-Container: "{{if ne (env "ENVIRONMENT") "production"}}{{containerName}}{{end}}"
      caddy.header.X-Image: "{{if ne (env "ENVIRONMENT") "production"}}{{imageName}}{{end}}"
```

### Dynamic Rate Limiting

```yaml
services:
  api:
    image: myapi:latest
    labels:
      caddy: api.example.com
      # Rate limiting based on container labels
      app.rate_limit: "100"
      app.rate_window: "1m"
      caddy.rate_limit: "{{label "app.rate_limit"}} {{label "app.rate_window"}}"
      caddy.reverse_proxy: "{{upstreams}}"
```

### Content-Based Routing

```yaml
services:
  frontend:
    image: react-app:latest
    volumes:
      - ./build:/app/build:ro
    labels:
      caddy: myapp.example.com
      # Serve React app for most routes
      caddy.root: "* {{mountSource "/app/build"}}"
      caddy.file_server: ""
      # Proxy API calls to backend
      caddy.route.0_@api.path: "/api/*"
      caddy.route.0_reverse_proxy: "@api backend:8080"
```

## Migration from Current Patterns

### Before (Limited to Reverse Proxying)

```yaml
labels:
  caddy: app.example.com
  caddy.reverse_proxy: "{{upstreams}}"
  caddy.tls: internal
```

### After (Full Caddy Feature Access)

```yaml
labels:
  caddy: app.example.com
  # Static files from volume
  caddy.root: "* {{mountSource "/app/public"}}"
  caddy.file_server: ""
  # Dynamic auth from environment
  caddy.basicauth: "/admin {{env "ADMIN_USER"}} {{env "ADMIN_PASS"}}"
  # Health-aware routing
  caddy.route.0_@api.path: "/api/*"
  caddy.route.0_reverse_proxy: "@api {{if isHealthy}}{{upstreams}}{{end}}"
  caddy.route.0_respond: "@api {{if not (isHealthy)}}503{{end}}"
  # Dynamic TLS
  caddy.tls.email: "{{env "TLS_EMAIL"}}"
```

## Best Practices

### Security Considerations

- **Environment variables**: Never expose sensitive environment variables in public logs
- **Volume access**: Validate mount paths exist and are readable
- **Label injection**: Sanitize label values that could contain malicious content

### Performance Considerations

- **Container inspection**: Some functions may require additional Docker API calls
- **Template complexity**: Keep templates simple for better performance and debugging
- **Caching**: Results are cached per configuration generation cycle

### Debugging

- **Template errors**: Will appear in Caddy logs with line numbers
- **Missing data**: Functions return empty strings for missing data
- **Type mismatches**: Use string conversion functions when needed

```yaml
# Debug template output
labels:
  caddy: debug.example.com
  caddy.respond: |
    200 {
      body "Debug: env={{env "DEBUG"}} mount={{mountSource "/app"}} running={{isRunning}}"
    }
```

## Implementation Notes

### Template Data Context

Templates have access to the full Docker container or service object:

```yaml
# Direct container access (advanced)
caddy.respond: |
  200 {
    body "Container ID: {{.ID}}"
  }

# Service access (Swarm mode)  
caddy.respond: |
  200 {
    body "Service: {{.Spec.Name}}"
  }
```

### Error Handling

- Functions return empty strings for missing data
- Invalid function calls are logged as warnings
- Template parsing errors prevent configuration generation

### Backward Compatibility

- All existing `{{upstreams}}` usage continues to work
- New functions are additive and optional
- No breaking changes to current label patterns