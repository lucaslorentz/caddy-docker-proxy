#!/bin/bash

set -e

if [ "${TRAVIS_PULL_REQUEST}" = "false" ] && [ "${TRAVIS_BRANCH}" = "master" ]; then
    echo "Deploying CI..."

    docker login -u lucaslorentz -p "$DOCKER_PASSWORD"
    docker push lucaslorentz/caddy-docker-proxy:ci
    docker push lucaslorentz/caddy-docker-proxy:ci-alpine
else
  echo "Skipping CI deploy"
fi
