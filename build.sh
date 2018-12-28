#!/bin/bash

set -e

echo ==PARAMETERS==
echo ARTIFACTS: "${ARTIFACTS:=./artifacts}"

dep ensure

go vet $(go list ./... | grep -v vendor)
go test -race -v $(go list ./... | grep -v vendor)

CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o ${ARTIFACTS}/binaries/linux/amd64/caddy
CGO_ENABLED=0 GOARCH=arm GOARM=6 GOOS=linux go build -o ${ARTIFACTS}/binaries/linux/arm32v6/caddy
CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -o ${ARTIFACTS}/binaries/windows/amd64/caddy.exe
