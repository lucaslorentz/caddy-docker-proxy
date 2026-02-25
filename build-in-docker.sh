#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
IMAGE="${GO_DOCKER_IMAGE:=golang:1.26-bookworm}"
ARTIFACTS="${ARTIFACTS:=./artifacts}"

mkdir -p "$ROOT_DIR/.cache/go-build" "$ROOT_DIR/.cache/gomod"

docker run --rm \
	-u "$(id -u):$(id -g)" \
	-e ARTIFACTS="$ARTIFACTS" \
	-e GOCACHE=/cache/go-build \
	-e GOMODCACHE=/cache/gomod \
	-v "$ROOT_DIR:/workspace" \
	-v "$ROOT_DIR/.cache/go-build:/cache/go-build" \
	-v "$ROOT_DIR/.cache/gomod:/cache/gomod" \
	-w /workspace \
	"$IMAGE" \
	bash -c "./build.sh"
