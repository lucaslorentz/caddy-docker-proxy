#!/bin/bash

set -e

docker buildx create --use
docker run --privileged --rm tonistiigi/binfmt --install all

find artifacts/binaries -type f -exec chmod +x {} \;

PLATFORMS="linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64"
OUTPUT=local
TAGS=
TAGS_ALPINE=

if [[ "${BUILD_SOURCEBRANCH}" == "refs/heads/master" ]]; then
    echo "Building and pushing CI images"

    docker login -u lucaslorentz -p "$DOCKER_PASSWORD"

    OUTPUT="type=registry"
    TAGS="-t lucaslorentz/caddy-docker-proxy:ci"
    TAGS_ALPINE="-t lucaslorentz/caddy-docker-proxy:ci-alpine"
fi

if [[ "${RELEASE_VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$ ]]; then
    echo "Releasing version ${RELEASE_VERSION}..."

    docker login -u lucaslorentz -p "$DOCKER_PASSWORD"

    PATCH_VERSION=$(echo $RELEASE_VERSION | cut -c2-)
    MINOR_VERSION=$(echo $PATCH_VERSION | cut -d. -f-2)

    OUTPUT="type=registry"
    TAGS="-t lucaslorentz/caddy-docker-proxy:latest \
        -t lucaslorentz/caddy-docker-proxy:${PATCH_VERSION} \
        -t lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}"
    TAGS_ALPINE="-t lucaslorentz/caddy-docker-proxy:alpine \
        -t lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}-alpine \
        -t lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}-alpine"
fi

docker buildx build -f Dockerfile . \
    -o $OUTPUT \
    --platform $PLATFORMS \
    $TAGS

docker buildx build -f Dockerfile-alpine . \
    -o $OUTPUT \
    --platform $PLATFORMS \
    $TAGS_ALPINE
