#!/bin/bash

set -e

glide install

go vet $(glide novendor)
go test -race -v $(glide novendor)

CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o caddy

docker build -t lucaslorentz/caddy-docker-proxy:ci -f Dockerfile .
docker build -t lucaslorentz/caddy-docker-proxy:ci-alpine -f Dockerfile-alpine .

if [ "${TRAVIS_PULL_REQUEST}" = "false" ] && [ "${TRAVIS_BRANCH}" = "master" ]; then
    docker login -u lucaslorentz -p "$DOCKER_PASSWORD"
    docker push lucaslorentz/caddy-docker-proxy:ci
    docker push lucaslorentz/caddy-docker-proxy:ci-alpine
fi
    
if [[ "${TRAVIS_TAG}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$ ]]; then
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
fi