#!/bin/bash

set -e

glide install

go vet $(glide novendor)
go test -race -v $(glide novendor)

CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o caddy
docker build -t lucaslorentz/caddy-docker-proxy:ci -f Dockerfile .
docker build -t lucaslorentz/caddy-docker-proxy:ci-alpine -f Dockerfile-alpine .
docker image tag lucaslorentz/caddy-docker-proxy:ci-alpine lucaslorentz/caddy-docker-proxy:test

CGO_ENABLED=0 GOARCH=arm GOARM=6 GOOS=linux go build -o caddy
docker build -t lucaslorentz/caddy-docker-proxy:ci-arm32v6 -f Dockerfile-arm32v6 .
docker build -t lucaslorentz/caddy-docker-proxy:ci-alpine-arm32v6 -f Dockerfile-alpine-arm32v6 .