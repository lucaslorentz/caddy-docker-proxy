#!/bin/bash

set -e

glide install

go vet $(glide novendor)
go test -race -v $(glide novendor)

CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o caddy

docker build -t lucaslorentz/caddy-docker-proxy:ci -f Dockerfile .
docker build -t lucaslorentz/caddy-docker-proxy:ci-alpine -f Dockerfile-alpine .
