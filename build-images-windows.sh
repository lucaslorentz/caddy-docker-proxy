#!/bin/bash

set -e

docker build -t lucaslorentz/caddy-docker-proxy:ci-nanoserver-1803 -f Dockerfile-nanoserver-1803 .
docker build -t lucaslorentz/caddy-docker-proxy:ci-nanoserver-1809 -f Dockerfile-nanoserver-1809 .
