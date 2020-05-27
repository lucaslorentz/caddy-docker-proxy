#!/bin/bash

set -e

echo ==PARAMETERS==
echo ARTIFACTS: "${ARTIFACTS:=./artifacts}"

cd plugin
go vet ./...
go test -race ./...
cd ../

go get -u github.com/caddyserver/xcaddy/cmd/xcaddy

CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
    xcaddy build \
    --output ${ARTIFACTS}/binaries/linux/amd64/caddy \
    --with github.com/lucaslorentz/caddy-docker-proxy/plugin/v2=$PWD/plugin

CGO_ENABLED=0 GOARCH=arm GOARM=6 GOOS=linux \
    xcaddy build \
    --output ${ARTIFACTS}/binaries/linux/arm32v6/caddy \
    --with github.com/lucaslorentz/caddy-docker-proxy/plugin/v2=$PWD/plugin

CGO_ENABLED=0 GOARCH=amd64 GOOS=windows \
    xcaddy build \
    --output ${ARTIFACTS}/binaries/windows/amd64/caddy.exe \
    --with github.com/lucaslorentz/caddy-docker-proxy/plugin/v2=$PWD/plugin
