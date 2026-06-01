#!/bin/bash

set -e

. ../functions.sh

CONFIG_PREFIX="cdp-test-caddy_test"

cleanup_configs() {
  docker config ls --format "{{.Name}}" | grep "^${CONFIG_PREFIX}-" | xargs -r docker config rm >/dev/null || true
}

cleanup_configs

docker stack deploy -c compose.yaml --prune caddy_test

retry bash -c "docker config ls --format '{{.Name}}' | grep '^${CONFIG_PREFIX}-'"

retry curl --show-error -s -k -f --resolve service.local:443:127.0.0.1 https://service.local ||
{
    echo "== Service errors =="
    docker service ps --no-trunc caddy_test_caddy --format "{{.Error}}"
    echo "== Caddy logs =="
    docker service logs caddy_test_caddy
    echo "== Controller logs =="
    docker service logs caddy_test_controller
    exit 1
}
