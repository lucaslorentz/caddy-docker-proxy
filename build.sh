#!/bin/sh

set -e

export GOARCH=amd64
export GOOS=linux

glide install

go build -o caddy

docker build -t lucaslorentz/caddy-docker-proxy .
docker push lucaslorentz/caddy-docker-proxy
