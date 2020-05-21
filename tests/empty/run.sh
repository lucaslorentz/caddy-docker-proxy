#!/bin/bash

set -e

docker stack deploy -c compose.yaml --prune caddy_test

sleep 2

docker service logs caddy_test_caddy
