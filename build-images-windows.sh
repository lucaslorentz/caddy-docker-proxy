#!/bin/bash

set -e

docker build -t lucaslorentz/caddy-docker-proxy:ci-nanoserver-1803 -f Dockerfile-nanoserver-1803 .
docker build -t lucaslorentz/caddy-docker-proxy:ci-no-telemetry-nanoserver-1803 -f Dockerfile-nanoserver-1803 --build-arg CADDY_EXEC=caddy-no-telemetry .
