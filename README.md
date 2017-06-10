# CADDY-DOCKER-PROXY

## Introduction
Caddy docker proxy is a caddy plugin that generates caddy config files from Docker Swarm Services metadata, making caddy act as a docker proxy.

## Supported metadata
# labels
caddy.address: list of addresses that should be proxied to that service
caddy.targetport: the port that is being served
caddy.targetpath: the path to add to the proxied urls
caddy.healthcheck: healthcheck url
caddy.websocket: set to enable websockets
caddy.basicauth: username password
caddy.tls: off
caddy.tls.dns: dns value

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