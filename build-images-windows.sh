#!/bin/bash

set -e

docker build -f Dockerfile-nanoserver . \
    --build-arg TARGETPLATFORM=windows/amd64 \
    --build-arg SERVERCORE_VERSION=1809 \
    --build-arg NANOSERVER_VERSION=1809 \
    -t lucaslorentz/caddy-docker-proxy:ci-nanoserver-1809

docker build -f Dockerfile-nanoserver . \
    --build-arg TARGETPLATFORM=windows/amd64 \
    --build-arg SERVERCORE_VERSION=ltsc2022 \
    --build-arg NANOSERVER_VERSION=ltsc2022 \
    -t lucaslorentz/caddy-docker-proxy:ci-nanoserver-ltsc2022

if [[ "${BUILD_SOURCEBRANCH}" == "refs/heads/master" ]]; then
    echo "Pushing CI images"
    
    docker login -u lucaslorentz -p "$DOCKER_PASSWORD"
    docker push lucaslorentz/caddy-docker-proxy:ci-nanoserver-1809
    docker push lucaslorentz/caddy-docker-proxy:ci-nanoserver-ltsc2022
fi

if [[ "${RELEASE_VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$ ]]; then
    echo "Releasing version ${RELEASE_VERSION}..."

    PATCH_VERSION=$(echo $RELEASE_VERSION | cut -c2-)
    MINOR_VERSION=$(echo $PATCH_VERSION | cut -d. -f-2)

    docker login -u lucaslorentz -p "$DOCKER_PASSWORD"

    # nanoserver-1809
    docker tag lucaslorentz/caddy-docker-proxy:ci-nanoserver-1809 lucaslorentz/caddy-docker-proxy:nanoserver-1809
    docker tag lucaslorentz/caddy-docker-proxy:ci-nanoserver-1809 lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}-nanoserver-1809
    docker tag lucaslorentz/caddy-docker-proxy:ci-nanoserver-1809 lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}-nanoserver-1809
    docker push lucaslorentz/caddy-docker-proxy:nanoserver-1809
    docker push lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}-nanoserver-1809
    docker push lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}-nanoserver-1809

    # nanoserver-ltsc2022
    docker tag lucaslorentz/caddy-docker-proxy:ci-nanoserver-ltsc2022 lucaslorentz/caddy-docker-proxy:nanoserver-ltsc2022
    docker tag lucaslorentz/caddy-docker-proxy:ci-nanoserver-ltsc2022 lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}-nanoserver-ltsc2022
    docker tag lucaslorentz/caddy-docker-proxy:ci-nanoserver-ltsc2022 lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}-nanoserver-ltsc2022
    docker push lucaslorentz/caddy-docker-proxy:nanoserver-ltsc2022
    docker push lucaslorentz/caddy-docker-proxy:${PATCH_VERSION}-nanoserver-ltsc2022
    docker push lucaslorentz/caddy-docker-proxy:${MINOR_VERSION}-nanoserver-ltsc2022
fi
