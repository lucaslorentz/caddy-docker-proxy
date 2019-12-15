#!/bin/bash

set -e

docker login -u lucaslorentz -p "$DOCKER_PASSWORD"

if [[ "${BUILD_SOURCEBRANCH}" == "refs/heads/master" ]]; then
    echo "Pushing CI images"
    docker push lucaslorentz/caddy-docker-proxy:ci
    docker push lucaslorentz/caddy-docker-proxy:ci-alpine
    docker push lucaslorentz/caddy-docker-proxy:ci-arm32v6
    docker push lucaslorentz/caddy-docker-proxy:ci-alpine-arm32v6
fi

if [[ "${RELEASE_VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$ ]]; then
    echo "Releasing version ${RELEASE_VERSION}..."

    PATCH_VERSION=$(echo $RELEASE_VERSION | cut -c2-)
    MINOR_VERSION=$(echo $PATCH_VERSION | cut -d. -f-2)

    # scratch
    docker tag lucaslorentz/caddy-docker-proxy:ci lucaslorentz/caddy-docker-proxy:latest
    docker tag lucaslorentz/caddy-docker-proxy:ci lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}
    docker tag lucaslorentz/caddy-docker-proxy:ci lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}
    docker push lucaslorentz/caddy-docker-proxy:latest
    docker push lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}
    docker push lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}

    # alpine
    docker tag lucaslorentz/caddy-docker-proxy:ci-alpine lucaslorentz/caddy-docker-proxy:alpine
    docker tag lucaslorentz/caddy-docker-proxy:ci-alpine lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}-alpine
    docker tag lucaslorentz/caddy-docker-proxy:ci-alpine lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}-alpine
    docker push lucaslorentz/caddy-docker-proxy:alpine
    docker push lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}-alpine
    docker push lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}-alpine

    # scratch arm32v6
    docker tag lucaslorentz/caddy-docker-proxy:ci-arm32v6 lucaslorentz/caddy-docker-proxy:latest-arm32v6
    docker tag lucaslorentz/caddy-docker-proxy:ci-arm32v6 lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}-arm32v6
    docker tag lucaslorentz/caddy-docker-proxy:ci-arm32v6 lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}-arm32v6
    docker push lucaslorentz/caddy-docker-proxy:latest-arm32v6
    docker push lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}-arm32v6
    docker push lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}-arm32v6

    # alpine arm32v6
    docker tag lucaslorentz/caddy-docker-proxy:ci-alpine-arm32v6 lucaslorentz/caddy-docker-proxy:alpine-arm32v6
    docker tag lucaslorentz/caddy-docker-proxy:ci-alpine-arm32v6 lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}-alpine-arm32v6
    docker tag lucaslorentz/caddy-docker-proxy:ci-alpine-arm32v6 lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}-alpine-arm32v6
    docker push lucaslorentz/caddy-docker-proxy:alpine-arm32v6
    docker push lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}-alpine-arm32v6
    docker push lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}-alpine-arm32v6
fi