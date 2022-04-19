#!/bin/bash

set -e

echo ==PARAMETERS==
echo ARTIFACTS: "${ARTIFACTS:=./artifacts}"

docker swarm init || true

if [ "$ARTIFACTS" != "./artifacts" ]
then
    mkdir -p ./artifacts/binaries/linux/amd64
    cp $ARTIFACTS/binaries/linux/amd64/caddy ./artifacts/binaries/linux/amd64/caddy
fi

docker build -q --build-arg TARGETPLATFORM=linux/amd64 -t caddy-docker-proxy:local -f Dockerfile-alpine .

(cd tests && . run.sh)
