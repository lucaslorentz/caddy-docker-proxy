#!/bin/bash

set -e

docker build -t lucaslorentz/caddy-docker-proxy:ci-nanoserver-1803 -f Dockerfile-nanoserver-1803 .
