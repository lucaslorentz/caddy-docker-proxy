# Caddy-Docker-Proxy 
[![Build Status](https://dev.azure.com/lucaslorentzlara/lucaslorentzlara/_apis/build/status/lucaslorentz.caddy-docker-proxy?branchName=master)](https://dev.azure.com/lucaslorentzlara/lucaslorentzlara/_build/latest?definitionId=1) [![Go Report Card](https://goreportcard.com/badge/github.com/lucaslorentz/caddy-docker-proxy)](https://goreportcard.com/report/github.com/lucaslorentz/caddy-docker-proxy)

## Introduction
This plugin enables Caddy to be used as a reverse proxy for Docker containers via labels.

## How does it work?
The plugin scans Docker metadata, looking for labels indicating that the service or container should be served by Caddy.

Then, it generates an in-memory Caddyfile with site entries and proxies pointing to each Docker service by their DNS name or container IP.

Every time a Docker object changes, the plugin updates the Caddyfile and triggers Caddy to gracefully reload, with zero-downtime.

## Table of contents

  * [Basic usage example, using docker-compose](#basic-usage-example-using-docker-compose)
  * [Labels to Caddyfile conversion](#labels-to-caddyfile-conversion)
    + [Tokens and arguments](#tokens-and-arguments)
    + [Ordering and isolation](#ordering-and-isolation)
    + [Sites, snippets and global options](#sites-snippets-and-global-options)
    + [Go templates](#go-templates)
  * [Template functions](#template-functions)
    + [upstreams](#upstreams)
  * [Reverse proxy examples](#reverse-proxy-examples)
  * [Docker configs](#docker-configs)
  * [Proxying services vs containers](#proxying-services-vs-containers)
    + [Services](#services)
    + [Containers](#containers)
  * [Execution modes](#execution-modes)
    + [Server](#server)
    + [Controller](#controller)
    + [Standalone (default)](#standalone-default)
  * [Caddy CLI](#caddy-cli)
  * [Docker images](#docker-images)
    + [Choosing the version numbers](#choosing-the-version-numbers)
    + [Chosing between default or alpine images](#chosing-between-default-or-alpine-images)
    + [CI images](#ci-images)
    + [ARM architecture images](#arm-architecture-images)
    + [Windows images](#windows-images)
    + [Custom images](#custom-images)
  * [Connecting to Docker Host](#connecting-to-docker-host)
  * [Volumes](#volumes)
  * [Trying it](#trying-it)
    + [With docker-compose file](#with-docker-compose-file)
    + [With run commands](#with-run-commands)
  * [Building it](#building-it)

## Basic usage example, using docker-compose
```shell
$ docker network create caddy
```

`caddy/docker-compose.yml`
```yml
version: "3.7"
services:
  caddy:
    image: lucaslorentz/caddy-docker-proxy:ci-alpine
    ports:
      - 80:80
      - 443:443
    environment:
      - CADDY_INGRESS_NETWORKS=caddy
    networks:
      - caddy
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - caddy_data:/data
    restart: unless-stopped

networks:
  caddy:
    external: true

volumes:
  caddy_data: {}
```
```shell
$ docker-compose up -d
```

`whoami/docker-compose.yml`
```yml
version: '3.7'
services:
  whoami:
    image: jwilder/whoami
    networks:
      - caddy
    labels:
      caddy: whoami.example.com
      caddy.reverse_proxy: "{{upstreams 8000}}"

networks:
  caddy:
    external: true
```
```shell
$ docker-compose up -d
```
Now, visit `https://whoami.example.com`. The site will be served [automatically over HTTPS](https://caddyserver.com/docs/automatic-https) with a certificate issued by Let's Encrypt or ZeroSSL.
		
## Labels to Caddyfile conversion
Please first read the [Caddyfile Concepts](https://caddyserver.com/docs/caddyfile/concepts) documentation to understand the structure of a Caddyfile.

Any label prefixed with `caddy` will be converted into a Caddyfile config, following these rules:

### Tokens and arguments

Keys are the directive name, and values are whitespace separated arguments:
```
caddy.directive: arg1 arg2
↓
{
	directive arg1 arg2
}
```

If you need whitespace or line-breaks inside one of the arguments, use double-quotes or backticks around it:
```
caddy.respond: / "Hello World" 200
↓
{
	respond / "Hello World" 200
}
```
```
caddy.respond: / `Hello\nWorld` 200
↓
{
	respond / `Hello
World` 200
}
```
```
caddy.respond: |
	/ `Hello
	World` 200
↓
{
	respond / `Hello
World` 200
}
```

Dots represent nesting, and grouping is done automatically:
```
caddy.directive: argA  
caddy.directive.subdirA: valueA  
caddy.directive.subdirB: valueB1 valueB2
↓
{
	directive argA {  
		subdirA valueA  
		subdirB valueB1 valueB2  
	}
}
```

Arguments for the parent directive are optional (e.g. no arguments to `directive`, setting subdirective `subdirA` directly):
```
caddy.directive.subdirA: valueA
↓
{
	directive {
		subdirA valueA
	}
}
```

Labels with empty values generate a directive without any arguments:
```
caddy.directive:
↓
{
	directive
}
```

### Ordering and isolation

Be aware that directives are subject to be sorted according to the default [directive order](https://caddyserver.com/docs/caddyfile/directives#directive-order) defined by Caddy, when the Caddyfile is parsed (after the Caddyfile is generated from labels).

[Directives](https://caddyserver.com/docs/caddyfile/directives) from labels are ordered alphabetically by default:
```
caddy.bbb: value
caddy.aaa: value
↓
{
	aaa value 
	bbb value
}
```

Suffix _&lt;number&gt; isolates directives that otherwise would be grouped:
```
caddy.route_0.a: value
caddy.route_1.b: value
↓
{
	route {
		a value
	}
	route {
		b value
	}
}
```

Prefix &lt;number&gt;_ isolates directives but also defines a custom ordering for directives (mainly relevant within [`route`](https://caddyserver.com/docs/caddyfile/directives/route) blocks), and directives without order prefix will go last:
```
caddy.1_bbb: value
caddy.2_aaa: value
caddy.3_aaa: value
↓
{
	bbb value
	aaa value
	aaa value
}
```

### Sites, snippets and global options

A label `caddy` creates a [site block](https://caddyserver.com/docs/caddyfile/concepts):
```
caddy: example.com
caddy.respond: "Hello World" 200
↓
example.com {
	respond "Hello World" 200
}
```

Or a [snippet](https://caddyserver.com/docs/caddyfile/concepts#snippets):
```
caddy: (encode)
caddy.encode: zstd gzip
↓
(encode) {
	encode zstd gzip
}
```

It's also possible to isolate Caddy configurations using suffix _&lt;number&gt;:
```
caddy_0: (snippet)
caddy_0.tls: internal
caddy_1: site-a.com
caddy_1.import: snippet
caddy_2: site-b.com
caddy_2.import: snippet
↓
(snippet) {
	tls internal
}
site_a {
	import snippet
}
site_b {
	import snippet
}
```

[Global options](https://caddyserver.com/docs/caddyfile/options) can be defined by not setting any value for `caddy`. They can be set in any container/service, including caddy-docker-proxy itself. [Here is an example](examples/standalone.yaml#L19)
```
caddy.email: you@example.com
↓
{
	email you@example.com
}
```

[Named matchers](https://caddyserver.com/docs/caddyfile/matchers#named-matchers) can be created using `@` inside labels:
```
caddy: localhost
caddy.@match.path: /sourcepath /sourcepath/*
caddy.reverse_proxy: @match localhost:6001
↓
localhost {
	@match {
		path /sourcepath /sourcepath/*
	}
	reverse_proxy @match localhost:6001
}
```

### Go templates

[Golang templates](https://golang.org/pkg/text/template/) can be used inside label values to increase flexibility. From templates, you have access to current Docker resource information. But, keep in mind that the structure that describes a Docker container is different from a service.

While you can access a service name like this:
```
caddy.respond: /info "{{.Spec.Name}}"
↓
respond /info "myservice"
```

The equivalent to access a container name would be:
```
caddy.respond: /info "{{index .Names 0}}"
↓
respond /info "mycontainer"
```

Sometimes it's not possile to have labels with empty values, like when using some UI to manage Docker. If that's the case, you can also use our support for go lang templates to generate empty labels.
```
caddy.directive: {{""}}
↓
directive
```

## Template functions

The following functions are available for use inside templates:

### upstreams

Returns all addresses for the current Docker resource separated by whitespace.

For services, that would be the service DNS name when **proxy-service-tasks** is **false**, or all running tasks IPs when **proxy-service-tasks** is **true**.

For containers, that would be the container IPs.

Only containers/services that are connected to Caddy ingress networks are used.

:warning: caddy docker proxy does a best effort to automatically detect what are the ingress networks. But that logic fails on some scenarios: [#207](https://github.com/lucaslorentz/caddy-docker-proxy/issues/207). To have a more resilient solution, you can manually configure Caddy ingress network using CLI option `ingress-networks` or environment variable `CADDY_INGRESS_NETWORKS`.

Usage: `upstreams [http|https] [port]`  

Examples:
```
caddy.reverse_proxy: {{upstreams}}
↓
reverse_proxy 192.168.0.1 192.168.0.2
```
```
caddy.reverse_proxy: {{upstreams https}}
↓
reverse_proxy https://192.168.0.1 https://192.168.0.2
```
```
caddy.reverse_proxy: {{upstreams 8080}}
↓
reverse_proxy 192.168.0.1:8080 192.168.0.2:8080
```
```
caddy.reverse_proxy: {{upstreams http 8080}}
↓
reverse_proxy http://192.168.0.1:8080 http://192.168.0.2:8080
```

:warning: Be carefull with quotes around upstreams. Quotes should only be added when using yaml. 
```
caddy.reverse_proxy: "{{upstreams}}"
↓
reverse_proxy "192.168.0.1 192.168.0.2"
```

## Reverse proxy examples
Proxying all requests to a domain to the container
```yml
caddy: example.com
caddy.reverse_proxy: {{upstreams}}
```

Proxying all requests to a domain to a subpath in the container
```yml
caddy: example.com
caddy.rewrite: * /target{path}
caddy.reverse_proxy: {{upstreams}}
```

Proxying requests matching a path, while stripping that path prefix
```yml
caddy: example.com
caddy.handle_path: /source/*
caddy.handle_path.0_reverse_proxy: {{upstreams}}
```

Proxying requests matching a path, rewriting to different path prefix
```yml
caddy: example.com
caddy.handle_path: /source/*
caddy.handle_path.0_rewrite: * /target{uri}
caddy.handle_path.1_reverse_proxy: {{upstreams}}
```

Proxying all websocket requests, and all requests to `/api*`, to the container
```yml
caddy: example.com
caddy.@ws.0_header: Connection *Upgrade*
caddy.@ws.1_header: Upgrade websocket
caddy.0_reverse_proxy: @ws {{upstreams}}
caddy.1_reverse_proxy: /api* {{upstreams}}
```

Proxying multiple domains, with certificates for each
```yml
caddy: example.com, example.org, www.example.com, www.example.org
caddy.reverse_proxy: {{upstreams}}
```

## Docker configs

> Note: This is for Docker Swarm only. Alternatively, use `CADDY_DOCKER_CADDYFILE_PATH` or `-caddyfile-path`

You can also add raw text to your Caddyfile using Docker configs. Just add Caddy label prefix to your configs and the whole config content will be inserted at the beginning of the generated Caddyfile, outside any server blocks.

[Here is an example](examples/standalone.yaml#L4)

## Proxying services vs containers
Caddy docker proxy is able to proxy to swarm services or raw containers. Both features are always enabled, and what will differentiate the proxy target is where you define your labels.

### Services
To proxy swarm services, labels should be defined at service level. In a docker-compose file, labels should be _inside_ `deploy`, like:
```yml
services:
  foo:
    deploy:
      labels:
        caddy: service.example.com
        caddy.reverse_proxy: {{upstreams}}
```

Caddy will use service DNS name as target or all service tasks IPs, depending on configuration **proxy-service-tasks**.

### Containers
To proxy containers, labels should be defined at container level. In a docker-compose file, labels should be _outside_ `deploy`, like:
```yml
services:
  foo:
    labels:
      caddy: service.example.com
      caddy.reverse_proxy: {{upstreams}}
```

## Execution modes

Each caddy docker proxy instance can be executed in one of the following modes.

### Server

Acts as a proxy to your Docker resources. The server starts without any configuration, and will not serve anything until it is configured by a "controller".

In order to make a server discoverable and configurable by controllers, you need to mark it with label `caddy_controlled_server` and define the controller network via CLI option `controller-network` or environment variable `CADDY_CONTROLLER_NETWORK`.

Server instances doesn't need access to Docker host socket and you can run it in manager or worker nodes.

[Configuration example](examples/distributed.yaml#L5)

### Controller

Controller monitors your Docker cluster, generates Caddy configuration and pushes to all servers it finds in your Docker cluster.

When controller instances are connected to more than one network, it is also necessary to define the controller network via CLI option `controller-network` or environment variable `CADDY_CONTROLLER_NETWORK`.

Controller instances require access to Docker host socket.

A single controller instance can configure all server instances in your cluster.

[Configuration example](examples/distributed.yaml#L21)

### Standalone (default)

This mode executes a controller and a server in the same instance and doesn't require additional configuration.

[Configuration example](examples/standalone.yaml#L11)

## Caddy CLI

This plugin extends caddy's CLI with the command `caddy docker-proxy`.

Run `caddy help docker-proxy` to see all available flags.

```
Usage of docker-proxy:
  -caddyfile-path string
        Path to a base Caddyfile that will be extended with Docker sites
  -controller-network string
        Network allowed to configure Caddy server in CIDR notation. Ex: 10.200.200.0/24
  -ingress-networks string
        Comma separated name of ingress networks connecting Caddy servers to containers.
        When not defined, networks attached to controller container are considered ingress networks
  -docker-sockets
        Comma separated docker sockets
        When not defined, DOCKER_HOST (or default docker socket if DOCKER_HOST not defined)
  -docker-certs-path
        Comma separated cert path, you could use empty value when no cert path for the concern index docker socket like cert_path0,,cert_path2
  -docker-apis-version
        Comma separated apis version, you could use empty value when no api version for the concern index docker socket like cert_path0,,cert_path2
  -label-prefix string
        Prefix for Docker labels (default "caddy")
  -mode
        Which mode this instance should run: standalone | controller | server
  -polling-interval duration
        Interval Caddy should manually check Docker for a new Caddyfile (default 30s)
  -process-caddyfile
        Process Caddyfile before loading it, removing invalid servers (default true)
  -proxy-service-tasks
        Proxy to service tasks instead of service load balancer (default true)
```

Those flags can also be set via environment variables:

```
CADDY_DOCKER_CADDYFILE_PATH=<string>
CADDY_CONTROLLER_NETWORK=<string>
CADDY_INGRESS_NETWORKS=<string>
CADDY_DOCKER_SOCKETS=<string>
CADDY_DOCKER_CERTS_PATH=<string>
CADDY_DOCKER_APIS_VERSION=<string>
CADDY_DOCKER_LABEL_PREFIX=<string>
CADDY_DOCKER_MODE=<string>
CADDY_DOCKER_POLLING_INTERVAL=<duration>
CADDY_DOCKER_PROCESS_CADDYFILE=<bool>
CADDY_DOCKER_PROXY_SERVICE_TASKS=<bool>
```

Check **examples** folder to see how to set them on a Docker Compose file.

## Docker images
Docker images are available at Docker hub:
https://hub.docker.com/r/lucaslorentz/caddy-docker-proxy/

### Choosing the version numbers
The safest approach is to use a full version numbers like 0.1.3.
That way you lock to a specific build version that works well for you.

But you can also use partial version numbers like 0.1. That means you will receive the most recent 0.1.x image. You will automatically receive updates without breaking changes.

### Chosing between default or alpine images
Our default images are very small and safe because they only contain Caddy executable.
But they're also quite hard to troubleshoot because they don't have shell or any other Linux utilities like curl or dig.

The alpine images variant are based on the Linux Alpine image, a very small Linux distribution with shell and basic utilities tools. Use `-alpine` images if you want to trade security and small size for a better troubleshooting experience.

### CI images
Images with the `ci` tag suffix means they were automatically generated by automated builds.
CI images reflect the current state of master branch and their stability is not guaranteed.
You may use CI images if you want to help testing the latest features before they're officially released.

### ARM architecture images
Currently we provide linux x86_64 images by default.

You can also find images for other architectures like `arm32v6` images that can be used on Raspberry Pi.

### Windows images
We recently introduced experimental windows containers images with the tag suffix `nanoserver-1803`.

Be aware that this needs to be tested further.

This is an example of how to mount the windows Docker pipe using CLI:
```shell
$ docker run --rm -it -v //./pipe/docker_engine://./pipe/docker_engine lucaslorentz/caddy-docker-proxy:ci-nanoserver-1803
```

### Custom images
If you need additional Caddy plugins, or need to use a specific version of Caddy, then you may use the `builder` variant of the [official Caddy Docker image](https://hub.docker.com/_/caddy) to make your own `Dockerfile`.

The main difference from the instructions on the official image is that you must override `CMD` to have the container run using the `caddy docker-proxy` command provided by this plugin.

```Dockerfile
ARG CADDY_VERSION=2.4.0
FROM caddy:${CADDY_VERSION}-builder AS builder

RUN xcaddy build \
    --with github.com/lucaslorentz/caddy-docker-proxy/plugin \
    --with <additional-plugins>

FROM caddy:${CADDY_VERSION}-alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy

CMD ["caddy", "docker-proxy"]
```

## Connecting to Docker Host
The default connection to Docker host varies per platform:
* At Unix: `unix:///var/run/docker.sock`
* At Windows: `npipe:////./pipe/docker_engine`

You can modify Docker connection using the following environment variables:

* **DOCKER_HOST**: to set the URL to the Docker server.
* **DOCKER_API_VERSION**: to set the version of the API to reach, leave empty for latest.
* **DOCKER_CERT_PATH**: to load the TLS certificates from.
* **DOCKER_TLS_VERIFY**: to enable or disable TLS verification; off by default.

## Volumes
On a production Docker swarm cluster, it's **very important** to store Caddy folder on persistent storage. Otherwise Caddy will re-issue certificates every time it is restarted, exceeding Let's Encrypt's quota.

To do that, map a persistent Docker volume to `/data` folder.

For resilient production deployments, use multiple Caddy replicas and map `/data` folder to a volume that supports multiple mounts, like Network File Sharing Docker volumes plugins.

Multiple Caddy instances automatically orchestrate certificate issuing between themselves when sharing `/data` folder.

## Trying it

### With docker-compose file

Clone this repository.

Deploy the compose file to swarm cluster:
```
$ docker stack deploy -c examples/standalone.yaml caddy-docker-demo
```

Wait a bit for services to startup...

Now you can access each service/container using different URLs
```
$ curl -k --resolve whoami0.example.com:443:127.0.0.1 https://whoami0.example.com
$ curl -k --resolve whoami1.example.com:443:127.0.0.1 https://whoami1.example.com
$ curl -k --resolve whoami2.example.com:443:127.0.0.1 https://whoami2.example.com
$ curl -k --resolve whoami3.example.com:443:127.0.0.1 https://whoami3.example.com
$ curl -k --resolve config.example.com:443:127.0.0.1 https://config.example.com
$ curl -k --resolve echo0.example.com:443:127.0.0.1 https://echo0.example.com/sourcepath/something
```

After testing, delete the demo stack:
```
$ docker stack rm caddy-docker-demo
```

### With run commands

```
$ docker run --name caddy -d -p 443:443 -v /var/run/docker.sock:/var/run/docker.sock lucaslorentz/caddy-docker-proxy:ci-alpine

$ docker run --name whoami0 -d -l caddy=whoami0.example.com -l "caddy.reverse_proxy={{upstreams 8000}}" -l caddy.tls=internal jwilder/whoami

$ docker run --name whoami1 -d -l caddy=whoami1.example.com -l "caddy.reverse_proxy={{upstreams 8000}}" -l caddy.tls=internal jwilder/whoami

$ curl -k --resolve whoami0.example.com:443:127.0.0.1 https://whoami0.example.com
$ curl -k --resolve whoami1.example.com:443:127.0.0.1 https://whoami1.example.com

$ docker rm -f caddy whoami0 whoami1
```

## Building it

You can build Caddy using [xcaddy](https://github.com/caddyserver/xcaddy) or [caddy docker builder](https://hub.docker.com/_/caddy).

Use module name **github.com/lucaslorentz/caddy-docker-proxy/plugin** to add this plugin to your build.
