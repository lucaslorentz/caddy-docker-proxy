#!/bin/bash

set -e

# Build
CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o artifacts/binaries/linux/amd64/caddy
chmod +x artifacts/binaries/linux/amd64/caddy
docker build -q -t caddy-docker-proxy:local -f Dockerfile-alpine .

(cd tests && ./run.sh)
