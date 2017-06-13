# CADDY-DOCKER-PROXY

## Introduction
Caddy docker proxy is a caddy plugin that generates caddy config files from Docker Swarm Services metadata, making caddy act as a docker proxy.

## Supported metadata
| Label        | Example           | Description  |
| -------------|-------------| -----|
| caddy.address | service.test.com | list of addresses that should be proxied to that service |
| caddy.targetport | 80 | the port being serverd by the service |
| caddy.targetpath | /api | the path being served by the service |
| caddy.websocket | (empty) | enable websocket proxxy |
| caddy.basicauth | username password | enable basic auth |
| caddy.tls | off | disable automatic TLS |
| caddy.tls.dns | route53 | use a dns provider for automatic TLS |

## Test it

Create caddy network:
```
docker network create --driver overlay caddy
```

Run caddy proxy:
```
docker service create --name caddy-docker-proxy --constraint=node.role==manager --publish 2015:2015 --mount type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock --network caddy lucaslorentz/caddy-docker-proxy -log stdout
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