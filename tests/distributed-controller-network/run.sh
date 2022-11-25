#!/bin/bash

set -e

. ../functions.sh

docker stack deploy -c compose.yaml --prune caddy_test

retry curl --show-error -s -k -f --resolve whoami_service.example.com:443:127.0.0.1 https://whoami_service.example.com &&
retry curl --show-error -s -k -f --resolve whoami_container.example.com:443:127.0.0.1 https://whoami_container.example.com || {
    docker service logs caddy_test_caddy_controller
    docker service logs caddy_test_caddy_server
    exit 1
}
