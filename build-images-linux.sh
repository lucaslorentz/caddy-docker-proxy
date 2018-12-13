#!/bin/bash

set -e

chmod +x artifacts/binaries/linux/amd64/caddy
docker build -t lucaslorentz/caddy-docker-proxy:ci -f Dockerfile .
docker build -t lucaslorentz/caddy-docker-proxy:ci-alpine -f Dockerfile-alpine .

chmod +x artifacts/binaries/linux/amd64/caddy-no-telemetry
docker build -t lucaslorentz/caddy-docker-proxy:ci-no-telemetry -f Dockerfile --build-arg CADDY_EXEC=caddy-no-telemetry .
docker build -t lucaslorentz/caddy-docker-proxy:ci-no-telemetry-alpine -f Dockerfile-alpine --build-arg CADDY_EXEC=caddy-no-telemetry .

chmod +x artifacts/binaries/linux/arm32v6/caddy
docker build -t lucaslorentz/caddy-docker-proxy:ci-arm32v6 -f Dockerfile-arm32v6 .
docker build -t lucaslorentz/caddy-docker-proxy:ci-alpine-arm32v6 -f Dockerfile-alpine-arm32v6 .

chmod +x artifacts/binaries/linux/arm32v6/caddy-no-telemetry
docker build -t lucaslorentz/caddy-docker-proxy:ci-no-telemetry-arm32v6 -f Dockerfile-arm32v6 --build-arg CADDY_EXEC=caddy-no-telemetry .
docker build -t lucaslorentz/caddy-docker-proxy:ci-no-telemetry-alpine-arm32v6 -f Dockerfile-alpine-arm32v6 --build-arg CADDY_EXEC=caddy-no-telemetry .
