# CADDY-DOCKER-PROXY [![Build Status](https://dev.azure.com/lucaslorentzlara/lucaslorentzlara/_apis/build/status/lucaslorentz.caddy-docker-proxy?branchName=master)](https://dev.azure.com/lucaslorentzlara/lucaslorentzlara/_build/latest?definitionId=1)

## Introduction
This plugin enables caddy to be used as a reverse proxy for Docker.

## How does it work?
It scans Docker metadata looking for labels indicating that the service or container should be exposed on caddy.

Then it generates an in memory Caddyfile with website entries and proxies directives pointing to each Docker service DNS name or container IP.

Every time a docker object changes, it updates the Caddyfile and triggers a caddy zero-downtime reload.

## Labels to Caddyfile conversion
Any label prefixed with caddy, will be converted to caddyfile configuration.

Label's keys are transformed into a directive name and its value becomes the directive arguments.

Example:
```
caddy.directive=valueA valueB
```

Generates:
```
directive valueA valueB
```

Dots inside labels keys represents neted levels inside caddyfile.

Example:  
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

Labels for parent directives are not required.

Example:
```
caddy.directive.subdirA=valueA
```

Generates:
```
directive {
	subdirA valueA
}
```

Labels with empty values generates directives without arguments.

Example:
```
caddy.directive=
```

Generates:
```
directive
```

Any _# suffix on labels is removed when generating caddyfile configuration, that allows you to write repeating directives with same name.

Example:
```
caddy.directive_1=value1
caddy.directive_2=value2
```

Generates:
```
directive value1
directive value2
```

### Automatic Proxy Generation
To automatically generate a website and a proxy directive pointing to a service or container, add the special label `caddy.address` to it.

Example:
```
caddy.address=service.example.com
```

Generates:
```
service.example.com {
	proxy / servicename
}
```

You can customize the automatic generated proxy with the following special labels:

| Label | Example | Description | Required |
| - | - | - | - |
| caddy.address | service.example.com | addresses that should be proxied separated by whitespace | Required |
| caddy.targetport | 80 | the port being server by container | Optional |
| caddy.targetpath | /api | the path being served by container | Optional |
| caddy.targetprotocol | https | the protocol being served by container | Optional |

When all the values above are added to a service, the following configuration will be generated:
```
service.example.com {
	proxy / https://servicename:80/api
}
```

### More examples
Proxying domain root to container root
```
caddy.address=service.example.com
```

Proxying domain root to container path
```
caddy.address=service.example.com
caddy.targetpath=/my-path
```

Proxying domain path to container root
```
caddy.address=service.example.com/path1
```

Proxying domain path to container path
```
caddy.address=service.example.com/path1
caddy.targetpath=/path2
```

Proxying multiple domains to container
```
caddy.address=service1.example.com service2.example.com
```

### Multiple caddyfile sections from one service/container
It's possible to generate multiple caddyfile sections for the same service/container by suffixing the caddy prefix with _#. That's usefull to expose multiple service ports at different urls.

Example:
```
caddy_0.address = portal.example.com
caddy_0.targetport = 80
caddy_1.address = admin.example.com
caddy_1.targetport = 81
```

Generates:
```
portal.example.com {
	proxy / servicename:80
}
admin.example.com {
	proxy / servicename:81
}
```

## Proxying services vs containers
Caddy docker proxy is able to proxy to swarm servcies or raw containers. Both features are always enabled, and what will differentiate the proxy target is where you define your labels.

### Services
To proxy swarm services, labels should be defined at service level. On a docker-compose file, that means labels should be inside deploy, like:
```
service:
  ...
  deploy:
    caddy.address=service.example.com
    caddy.targetport=80
```

Caddy will use service dns name as target, swarm takes care of load balancing into all containers of that service.

### Containers
To proxy containers, labels should be defined at container level. On a docker-compose file, that means labels should be outside deploy, like:
```
service:
  ...
  caddy.address=service.example.com
  caddy.targetport=80
```
When proxying a container, caddy uses a single container IP as target. Currently multiple containers/replicas are not supported under the same website.

## Docker configs
You can also add raw text to your caddyfile using docker configs. Just add caddy label prefix to your configs and the whole config content will be prepended to the generated caddyfile.

Example at:  
https://github.com/lucaslorentz/caddy-docker-proxy/blob/master/examples/service-proxy.yaml#L4  
https://github.com/lucaslorentz/caddy-docker-proxy/blob/master/examples/Caddyfile

## Docker images
Docker images are available at Docker hub:
https://hub.docker.com/r/lucaslorentz/caddy-docker-proxy/

### Choosing the version numbers
The safest approach is to use a full version numbers like 0.1.3.
That way you lock to a specific build version that works well for you.

But you can also use partial version numbers like 0.1. That means you will receive the most recent 0.1.x image. You will automatically receive updates without breaking changes.

### Chosing between default or alpine images
Our default images are very small and safe because they only contain caddy executable.
But they're also quite hard to throubleshoot because they don't have shell or any other Linux utilities like curl or dig.

The alpine images variant are based on Linux Alpine image, a very small Linux distribution with shell and basic utilities tools. Use `-alpine` images if you want to trade security and small size for better throubleshooting experience.

### CI images
Images with `ci` on it's tag name means they was automatically generated by automated builds.
CI images reflect the current state of master branch and they might be very broken sometimes.
You should use CI images if you want to help testing latest features before they're officialy released.

### ARM architecture images
Currently we provide linux x86_64 images by default.

You can also find images for other architectures like `arm32v6` images that can be used on Raspberry Pi.

### Windows images
We recently introduced experimental windows containers images with suffix `nanoserver-1803`.

Be aware that this needs to be tested further.

This is an example of how to mount the windows docker pipe using CLI:
```
docker run --rm -it -p 2015:2015 -v //./pipe/docker_engine://./pipe/docker_engine lucaslorentz/caddy-docker-proxy:ci-nanoserver-1803 -agree -email email@example.com -log stdout
```

## Caddy CLI
All flags and environment variables supported by Caddy CLI are also supported:
https://caddyserver.com/docs/cli

Check **examples** folder to see how to set them on a docker compose file.

This plugin provides these flags:

```
-docker-label-prefix string
      Prefix for Docker labels (default "caddy")
-docker-polling-interval duration
      Interval caddy should manually check docker for a new caddyfile (default 30s)
-proxy-service-tasks
      Proxy to service tasks instead of VIP
```

Those flags can also be set via environment variables:

```
CADDY_DOCKER_LABEL_PREFIX=<string>
CADDY_DOCKER_POLLING_INTERVAL=<duration>
CADDY_DOCKER_PROXY_SERVICE_TASKS=<bool>
```

## Connecting to Docker Host
The default connection to docker host varies per platform:
* At Unix: `unix:///var/run/docker.sock`
* At Windows: `npipe:////./pipe/docker_engine`

You can modify docker connection using the following environment variables:

* **DOCKER_HOST**: to set the url to the docker server.
* **DOCKER_API_VERSION**: to set the version of the API to reach, leave empty for latest.
* **DOCKER_CERT_PATH**: to load the tls certificates from.
* **DOCKER_TLS_VERIFY**: to enable or disable TLS verification, off by default.

## Volumes
On a production docker swarm cluster, it's **very important** to store Caddy folder on a persistent storage. Otherwise Caddy will re-issue certificates every time it is restarted, exceeding let's encrypt quota.

To do that map a docker volume to `/root/.caddy` folder.

Since Caddy version 0.10.11, it is possible to run multiple caddy instances sharing same certificates.

For resilient production deployments, use multiple caddy replicas and map a`/root/.caddy` folder to a volume that supports multiple mounts, like Network File Sharing docker volumes plugins.

[Here is an example](examples/efs-volume.yaml) of compose file with replicas and persistent volume using  Rexray EFS Plugin for AWS.

## Trying it

Clone this repository.

Deploy the compose file to swarm cluster:
```
docker stack deploy -c examples/service-proxy.yaml caddy-docker-demo
```

Wait a bit for services startup...

Now you can access both services using different urls
```
curl -H Host:whoami0.example.com http://localhost:2015

curl -H Host:whoami1.example.com http://localhost:2015

curl -H Host:config.example.com http://localhost:2015
```

After testing, delete the demo stack:
```
docker stack rm caddy-docker-demo
```

## Building it
You can use our caddy build wrapper **build.sh** and include additional plugins on https://github.com/lucaslorentz/caddy-docker-proxy/blob/master/main.go#L5

Or, you can build from caddy repository and import  **caddy-docker-proxy** plugin on file https://github.com/mholt/caddy/blob/master/caddy/caddymain/run.go :
```
import (
  _ "github.com/lucaslorentz/caddy-docker-proxy/plugin"
)
```
