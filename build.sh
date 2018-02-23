#!/bin/bash

set -e

glide install

go vet $(glide novendor)
go test -race -v $(glide novendor)

CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o caddy

docker build -t lucaslorentz/caddy-docker-proxy:ci .

if [ "${TRAVIS_PULL_REQUEST}" = "false" ] && [ "${TRAVIS_BRANCH}" = "master" ]; then
    docker login lucaslorentz $DOCKER_PASSWORD
    docker push lucaslorentz/caddy-docker-proxy:ci
fi
    
if [[ "${TRAVIS_TAG}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    export VERSION=$(echo $TAG | cut -c2-)
    docker login lucaslorentz $DOCKER_PASSWORD
    docker tag lucaslorentz/caddy-docker-proxy:ci lucaslorentz/caddy-docker-proxy:latest
    docker tag lucaslorentz/caddy-docker-proxy:ci lucaslorentz/caddy-docker-proxy:$VERSION
    docker push lucaslorentz/caddy-docker-proxy:latest
    docker push lucaslorentz/caddy-docker-proxy:$VERSION
fi