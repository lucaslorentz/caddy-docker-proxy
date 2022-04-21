#!/bin/bash

set -e

echo ==PARAMETERS==
echo ARTIFACTS: "${ARTIFACTS:=./artifacts}"

if [ "$ARTIFACTS" != "./artifacts" ]
then
    mkdir -p ./artifacts/binaries/linux/amd64
    cp $ARTIFACTS/binaries/linux/amd64/caddy ./artifacts/binaries/linux/amd64/caddy
fi

find artifacts/binaries -type f -exec chmod +x {} \;

docker build -q -f Dockerfile-alpine . \
    --build-arg TARGETPLATFORM=linux/amd64 \
    -t caddy-docker-proxy:local

docker swarm init || true

export DOCKER_SOCKET_PATH="/var/run/docker.sock"
export DOCKER_SOCKET_TYPE="bind"

(cd tests && . run.sh)
