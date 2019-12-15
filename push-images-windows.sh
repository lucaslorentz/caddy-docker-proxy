#!/bin/bash

set -e

docker login -u lucaslorentz -p "$DOCKER_PASSWORD"

if [[ "${BUILD_SOURCEBRANCH}" == "refs/heads/master" ]]; then
    echo "Pushing CI images"
    docker push lucaslorentz/caddy-docker-proxy:ci-nanoserver-1803
fi

if [[ "${RELEASE_VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$ ]]; then
    echo "Releasing version ${RELEASE_VERSION}..."

    PATCH_VERSION=$(echo $RELEASE_VERSION | cut -c2-)
    MINOR_VERSION=$(echo $PATCH_VERSION | cut -d. -f-2)

    docker login -u lucaslorentz -p "$DOCKER_PASSWORD"

    # nanoserver-1803
    docker tag lucaslorentz/caddy-docker-proxy:ci-nanoserver-1803 lucaslorentz/caddy-docker-proxy:nanoserver-1803
    docker tag lucaslorentz/caddy-docker-proxy:ci-nanoserver-1803 lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}-nanoserver-1803
    docker tag lucaslorentz/caddy-docker-proxy:ci-nanoserver-1803 lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}-nanoserver-1803
    docker push lucaslorentz/caddy-docker-proxy:nanoserver-1803
    docker push lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}-nanoserver-1803
    docker push lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}-nanoserver-1803
fi
