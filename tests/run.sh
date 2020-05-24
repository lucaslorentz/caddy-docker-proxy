#!/bin/bash

set -e

for d in */
do
  (cd $d && . run.sh)
done

docker stack rm caddy_test
