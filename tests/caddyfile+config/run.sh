#!/bin/bash

set -e

. ../functions.sh

docker stack deploy -c compose.yaml --prune caddy_test

retry curl -k -f --resolve whoami0.local:443:127.0.0.1 https://whoami0.local/whoami &&
retry curl -k -f --resolve whoami0.local:443:127.0.0.1 https://whoami0.local/caddyfile &&
retry curl -k -f --resolve whoami0.local:443:127.0.0.1 https://whoami0.local/config ||
{
    docker service logs caddy_test_caddy
    exit 1
}
