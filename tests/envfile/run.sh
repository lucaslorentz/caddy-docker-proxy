#!/bin/bash

set -e

. ../functions.sh

docker stack deploy -c compose.yaml --prune caddy_test

retry curl --show-error -s -k -f --resolve service.local:443:127.0.0.1 https://service.local/testenv | grep "Hello from TestEnv" || {
    docker service logs caddy_test_caddy
    exit 1
}
