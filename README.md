# CADDY-DOCKER-PROXY [![Build Status](https://travis-ci.org/lucaslorentz/caddy-docker-proxy.svg?branch=master)](https://travis-ci.org/lucaslorentz/caddy-docker-proxy)

## Introduction
This plugin enables caddy to be used as a reverse proxy for Docker Swarm services.

## How does it work?
It scans Docker Swarm services metadata looking for labels indicating that the service should be exposed on caddy.

Then it generates an in memory Caddyfile with website entries and proxies directives pointing to each Docker Service internal DNS name.

Every time Docker services changes, it updates the Caddyfile and triggers a caddy zero-downtime reload.

## Basic service labels
To expose a service inside caddy configuration, you just need to add labels starting with caddy to the service.

Those are the main labels that configures the basic behavior of the proxying:

| Label | Example | Description |
| - | - | - |
| caddy.address | service.example.com | addresses that should be proxied to that service separated by ',' |
| caddy.targetport | 80 | the port being serverd by the service |
| caddy.targetpath | /api | the path being served by the service |

### Usage examples
Proxying domain root to container root
```
caddy.address=service.example.com
caddy.targetport=80
```

Proxying domain root to container path
```
caddy.address=service.example.com
caddy.targetport=80
caddy.targetpath=/my-path
```

Proxying domain path to container root
```
caddy.address=service.example.com/path1
caddy.targetport=80
```

Proxying domain path to container path
```
caddy.address=service.example.com/path1
caddy.targetport=80
caddy.targetpath=/path2
```

Proxying multiple domains to container
```
caddy.address=service1.example.com,service2.example.com
caddy.targetport=80
```
## More service labels
Any other label prefixed with caddy, will also be converted to caddyfile configuration based on the following rules:

Label's keys are transformed into a directive and its value becomes the directive arguments. Example:
```
caddy.directive=valueA valueB
```
Generates:
```
directive valueA valueB
```

Dots represents nested directives. Example:
```
caddy.directive=argA
caddy.directive.subdirA=valueA
caddy.directive.subdirB=valueB1 valueB2
```
Generates:
```
directive argA {
	subdirA valueA
	subdirB valueB1 valueB2
}
```

Labels for parent directives are not required. Example:
```
caddy.directive.subdirA=valueA
```
Generates:
```
directive {
	subdirA valueA
}
```

Labels with empty values generates directives without arguments. Example:
```
caddy.directive=
```
Generates:
```
directive
```

It's possible to add directives to the automatically created proxy directive. Example:
```
caddy.proxy.websocket=
```
Generates:
```
service.example.com {
	proxy / servicedns:80/api {
		websocket
	}
}
```

Any _* suffix on labels is removed when generating caddyfile configuration, that allows you to write repeating directives. Example:
```
caddy.directive_1=value1
caddy.directive_2=value2
```
Generates:
```
directive value1
directive value2
```

## Multiple caddyfile sections from one service
It's possible to generate multiple caddyfile sections for the same service by suffixing the caddy prefix with _*. That's usefull to expose multiple service ports at different urls.

For example:
```
caddy_0.address = portal.example.com
caddy_0.targetport = 80
caddy_1.address = admin.example.com
caddy_1.targetport = 81
```
Generates:
```
portal.example.com {
	proxy / servicedns:80
}
admin.example.com {
	proxy / servicedns:81
}
```

## Building it
You can use our caddy build wrapper **build.sh** and include additional plugins on https://github.com/lucaslorentz/caddy-docker-proxy/blob/master/main.go#L5

Or, you can build from caddy repository and import  **caddy-docker-proxy** plugin on file https://github.com/mholt/caddy/blob/master/caddy/caddymain/run.go :
```
import (
  _ "github.com/lucaslorentz/caddy-docker-proxy/plugin"
)
```

## Docker images
Docker images are available at Docker Registry:
https://hub.docker.com/r/lucaslorentz/caddy-docker-proxy/

## Configuring connection to Docker Server

The following environment variables are supported to configure connection with Docker host.

* **DOCKER_HOST**: to set the url to the docker server.
* **DOCKER_API_VERSION**: to set the version of the API to reach, leave empty for latest.
* **DOCKER_CERT_PATH**: to load the tls certificates from.
* **DOCKER_TLS_VERIFY**: to enable or disable TLS verification, off by default.

In case you see error messages like `client version 1.37 is too new. Maximum supported API version is 1.35`. Set the environment variable DOCKER_API_VERSION to the maximum supported API version before connecting.


## Trying it

Clone this repository.

Deploy the compose file to swarm cluster:
```
docker stack deploy -c examples/demo.yaml caddy-docker-demo
```

Wait a bit for services startup...

Now you can access both services using different urls
```
curl -H Host:whoami0.caddy-docker-demo http://localhost:2015

curl -H Host:whoami1.caddy-docker-demo http://localhost:2015
```

After testing, delete the demo stack:
```
docker stack rm caddy-docker-demo
```