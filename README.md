# CADDY-DOCKER-PROXY

## Introduction
Caddy docker proxy is a caddy plugin that generates caddy config files from Docker Swarm Services metadata, making caddy act as a docker proxy.

## Labels for service
| Label        | Example           | Description  |
| -------------|-------------| -----|
| caddy.address | service.test.com | list of addresses that should be proxied to that service |
| caddy.targetport | 80 | the port being serverd by the service |
| caddy.targetpath | /api | the path being served by the service |

TODO: Describe automatic label to directive mapping

## Test it

Create caddy network:
```
docker network create --driver overlay caddy
```

Run caddy proxy:
```
docker service create --name docker-caddy --constraint=node.role==manager --publish 2015:2015 --mount type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock --network caddy lucaslorentz/docker-caddy -log stdout
```

Create services:
```
docker service create --network caddy -l caddy.address=whoami0.caddy-proxy -l caddy.targetport=8000 -l caddy.tls=off --name whoami0 jwilder/whoami


docker service create --network caddy -l caddy.address=whoami1.caddy-proxy -l caddy.targetport=8000 -l caddy.tls=off --name whoami1 jwilder/whoami
```

Access them through the proxy:
```
curl -H Host:whoami0.caddy-proxy http://localhost:2015

curl -H Host:whoami1.caddy-proxy http://localhost:2015
```