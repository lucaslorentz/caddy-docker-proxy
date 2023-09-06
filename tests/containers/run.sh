#!/bin/bash

set -e

. ../functions.sh

trap "docker rm -f caddy whoami0 whoami1 whoami_stopped" EXIT

{
    docker run --name caddy -d -p 4443:443 -e CADDY_DOCKER_SCAN_STOPPED_CONTAINERS=true -v /var/run/docker.sock:/var/run/docker.sock caddy-docker-proxy:local &&
    docker run --name whoami0 -d -l caddy=whoami0.example.com -l "caddy.reverse_proxy={{upstreams 80}}" -l caddy.tls=internal containous/whoami &&
    docker run --name whoami1 -d -l caddy=whoami1.example.com -l "caddy.reverse_proxy={{upstreams 80}}" -l caddy.tls=internal containous/whoami &&
    docker create --name whoami_stopped -l caddy=whoami_stopped.example.com -l "caddy.respond=\"I'm a stopped container!\" 200" -l caddy.tls=internal containous/whoami &&

    retry curl -k --resolve whoami0.example.com:4443:127.0.0.1 https://whoami0.example.com:4443 &&
    retry curl -k --resolve whoami1.example.com:4443:127.0.0.1 https://whoami1.example.com:4443 &&
    retry curl -k --resolve whoami_stopped.example.com:4443:127.0.0.1 https://whoami_stopped.example.com:4443
} || {
    echo "Test failed"
    exit 1
}