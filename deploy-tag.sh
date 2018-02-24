#!/bin/bash

set -e

if [[ "${TRAVIS_TAG}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$ ]]; then
    echo "Deploying version ${TRAVIS_TAG}..."

    export PATCH_VERSION=$(echo $TRAVIS_TAG | cut -c2-)
    export MINOR_VERSION=$(echo $PATCH_VERSION | cut -d. -f-2)

    docker login -u lucaslorentz -p "$DOCKER_PASSWORD"

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
else
  echo "Skipping version deploy"
fi