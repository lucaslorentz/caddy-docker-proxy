#!/bin/bash

set -e

echo ==PARAMETERS==
echo ARTIFACTS: "${ARTIFACTS:=./artifacts}"

if [ "$ARTIFACTS" != "./artifacts" ]
then
    mkdir -p ./artifacts/binaries/windows/amd64
    cp $ARTIFACTS/binaries/windows/amd64/caddy ./artifacts/binaries/windows/amd64/caddy
fi

env
exit 1

docker build -q -f Dockerfile-nanoserver . \
    --build-arg TARGETPLATFORM=windows/amd64 \
    --build-arg SERVERCORE_VERSION=ltsc2022 \
    --build-arg NANOSERVER_VERSION=ltsc2022 \
    -t caddy-docker-proxy:local

docker swarm init --advertise-addr 127.0.0.1 || true

export DOCKER_SOCKET_PATH='\\.\pipe\docker_engine'
export DOCKER_SOCKET_TYPE="npipe"

(cd tests && . run.sh)
