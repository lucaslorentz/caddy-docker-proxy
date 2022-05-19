#!/bin/bash

set -e

echo ==PARAMETERS==
echo ARTIFACTS: "${ARTIFACTS:=./artifacts}"

cd plugin
go vet ./...
go test -race ./...
cd ../

go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest

# AMD64
CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
    xcaddy build \
    --output ${ARTIFACTS}/binaries/linux/amd64/caddy \
    --with github.com/lucaslorentz/caddy-docker-proxy/v2=$PWD

# ARM
CGO_ENABLED=0 GOARCH=arm GOARM=6 GOOS=linux \
    xcaddy build \
    --output ${ARTIFACTS}/binaries/linux/arm/v6/caddy \
    --with github.com/lucaslorentz/caddy-docker-proxy/v2=$PWD

CGO_ENABLED=0 GOARCH=arm GOARM=7 GOOS=linux \
    xcaddy build \
    --output ${ARTIFACTS}/binaries/linux/arm/v7/caddy \
    --with github.com/lucaslorentz/caddy-docker-proxy/v2=$PWD

CGO_ENABLED=0 GOARCH=arm64 GOOS=linux \
    xcaddy build \
    --output ${ARTIFACTS}/binaries/linux/arm64/caddy \
    --with github.com/lucaslorentz/caddy-docker-proxy/v2=$PWD

# AMD64 WINDOWS
CGO_ENABLED=0 GOARCH=amd64 GOOS=windows \
    xcaddy build \
    --output ${ARTIFACTS}/binaries/windows/amd64/caddy.exe \
    --with github.com/lucaslorentz/caddy-docker-proxy/v2=$PWD
