#!/bin/bash

set -e

for d in */
do
  echo ""
  echo ""
  echo "=== Running test $d ==="
  echo ""
  (cd $d && . run.sh)
done

docker stack rm caddy_test
