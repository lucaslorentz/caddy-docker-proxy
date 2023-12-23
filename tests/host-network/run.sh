#!/bin/bash

set -e

. ../functions.sh

docker stack deploy -c compose.yaml --prune caddy_test

retry curl --show-error -s -k -f --resolve whoami0.example.com:4443:127.0.0.1 https://whoami0.example.com:4443 &&
    curl --show-error -s -k -f --resolve whoami1.example.com:4443:127.0.0.1 https://whoami1.example.com:4443 || {
    docker service logs caddy_test_caddy_controller
    docker service logs caddy_test_caddy_server
    exit 1
}
