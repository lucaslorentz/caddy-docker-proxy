#!/bin/bash

set -e

wait_stack_removed() {
  for _ in $(seq 1 30); do
    remaining=$(docker network ls --filter label=com.docker.stack.namespace=caddy_test -q)
    if [ -z "$remaining" ]; then
      return 0
    fi
    sleep 1
  done
  echo "warning: caddy_test stack networks still present after 30s" >&2
}

trap "exit 1" INT
trap "docker stack rm caddy_test; wait_stack_removed" EXIT

docker network create --driver overlay --attachable caddy_test || true

for d in */
do
  docker stack rm caddy_test || true
  wait_stack_removed

  echo ""
  echo ""
  echo "=== Running test $d ==="
  echo ""
  (cd $d && . run.sh)
done
