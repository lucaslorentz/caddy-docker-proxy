# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Caddy-Docker-Proxy is a Caddy v2 plugin that enables dynamic reverse proxy configuration for Docker containers and services using Docker labels. The plugin monitors Docker events and automatically generates Caddyfiles to proxy requests to containerized applications.

## Core Architecture

### Main Components

- **Module Registration** (`module.go`): Registers the plugin with Caddy
- **Command Interface** (`cmd.go`): Defines the `caddy docker-proxy` CLI command and configuration options
- **Docker Loader** (`loader.go`): Core service that monitors Docker events and manages configuration updates
- **Generator Package** (`generator/`): Converts Docker labels into Caddyfile configurations
- **Caddyfile Package** (`caddyfile/`): Handles Caddyfile parsing, processing, and generation
- **Docker Package** (`docker/`): Abstracts Docker API interactions
- **Config Package** (`config/`): Manages plugin configuration options

### Execution Modes

The plugin supports three execution modes:
- **Standalone** (default): Controller + Server in same instance
- **Controller**: Monitors Docker and pushes config to servers
- **Server**: Receives configuration from controllers and serves traffic

### Label Processing Flow

1. Docker events trigger configuration updates
2. Generator scans containers/services for `caddy` labels
3. Labels are parsed and converted to Caddyfile directives
4. Caddyfile is validated and converted to JSON config
5. Configuration is applied (standalone) or pushed to servers (distributed)

## Development Commands

### Building

```bash
# Standard build script
./build.sh

# Manual build with xcaddy
go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest
xcaddy build --with github.com/lucaslorentz/caddy-docker-proxy/v2=$PWD
```

### Testing

```bash
# Run Go tests
go test -race ./...

# Run integration tests (requires Docker)
./tests/run.sh

# Run specific platform tests
./run-docker-tests-linux.sh    # Linux containers
./run-docker-tests-windows.sh  # Windows containers
```

### Code Quality

```bash
# Lint and vet
go vet ./...
```

## Key Configuration Options

All CLI flags can be set via environment variables with `CADDY_DOCKER_` prefix:

- `--mode`: standalone|controller|server
- `--docker-sockets`: Comma-separated Docker socket paths
- `--ingress-networks`: Networks connecting Caddy to containers
- `--label-prefix`: Docker label prefix (default: "caddy")
- `--polling-interval`: Manual check interval (default: 30s)
- `--proxy-service-tasks`: Proxy to service tasks vs load balancer
- `--process-caddyfile`: Validate and process generated Caddyfile

## Label System

Docker labels use dot notation to create nested Caddyfile structures:
- `caddy: example.com` → Site block
- `caddy.reverse_proxy: {{upstreams}}` → Directive with template
- `caddy.@matcher.path: /api/*` → Named matcher
- `caddy_0`, `caddy_1` → Multiple isolated configurations

## Template Functions

- `{{upstreams [protocol] [port]}}`: Returns container/service addresses
- Access to Docker container/service metadata in templates

## File Organization

- `/caddyfile/`: Caddyfile processing and generation logic
- `/generator/`: Core label-to-config conversion
- `/docker/`: Docker client abstraction and utilities  
- `/tests/`: Integration test scenarios
- `/examples/`: Docker Compose usage examples
- `/config/`: Configuration structures and validation

## Testing Structure

Integration tests in `/tests/` use Docker Compose scenarios:
- Each subdirectory tests a specific feature/configuration
- `run.sh` scripts automate test execution
- Tests verify generated Caddyfile and proxy behavior

## Common Development Patterns

- Use `logger()` function for consistent logging with zap
- Docker clients are abstracted through the `docker` package
- Configuration changes trigger graceful Caddy reloads
- Event throttling prevents excessive updates during Docker operations
- Template processing allows dynamic configuration based on container state