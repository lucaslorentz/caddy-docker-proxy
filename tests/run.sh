#!/bin/bash

set -e

docker network create --driver overlay --attachable caddy_test || true

for d in */
do
  docker stack rm caddy_test || true

  echo ""
  echo ""
  echo "=== Running test $d ==="
  echo ""
  (cd $d && . run.sh)
done

docker stack rm caddy_test