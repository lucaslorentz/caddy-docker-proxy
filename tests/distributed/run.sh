#!/bin/bash

set -e

. ../functions.sh

docker stack deploy -c compose.yaml --prune caddy_test

retry curl -k -f --resolve whoami0.example.com:443:127.0.0.1 https://whoami0.example.com &&
retry curl -k -f --resolve whoami1.example.com:443:127.0.0.1 https://whoami1.example.com &&
retry curl -k -f --resolve whoami2.example.com:443:127.0.0.1 https://whoami2.example.com &&
retry curl -k -f --resolve whoami3.example.com:443:127.0.0.1 https://whoami3.example.com &&
retry curl -k -f --resolve echo0.example.com:443:127.0.0.1 https://echo0.example.com/sourcepath/something || {
    docker service logs caddy_test_caddy_controller
    docker service logs caddy_test_caddy_server
    exit 1
}
