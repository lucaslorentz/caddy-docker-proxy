#!/bin/bash

set -e

. ../functions.sh

docker stack deploy -c compose.yaml --prune caddy_test

retry curl --show-error -s -k -f --resolve service.local:443:127.0.0.1 https://service.local/caddyfile | grep caddyfile &&
retry curl --show-error -s -k -f --resolve service.local:443:127.0.0.1 https://service.local/config | grep config &&
retry curl --show-error -s -k -f --resolve caddyfile.local:443:127.0.0.1 https://caddyfile.local | grep caddyfile &&
retry curl --show-error -s -k -f --resolve config.local:443:127.0.0.1 https://config.local | grep config ||
{
    echo "== Service errors =="
    docker service ps --no-trunc caddy_test_caddy --format "{{.Error}}"
    echo "== Service logs =="
    docker service logs caddy_test_caddy
    exit 1
}
