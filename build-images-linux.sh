#!/bin/bash

set -e

docker build -t lucaslorentz/caddy-docker-proxy:ci -f Dockerfile .
docker build -t lucaslorentz/caddy-docker-proxy:ci-alpine -f Dockerfile-alpine .

docker build -t lucaslorentz/caddy-docker-proxy:ci-arm32v6 -f Dockerfile-arm32v6 .
docker build -t lucaslorentz/caddy-docker-proxy:ci-alpine-arm32v6 -f Dockerfile-alpine-arm32v6 .
