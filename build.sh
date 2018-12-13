#!/bin/bash

set -e

echo ==PARAMETERS==
echo ARTIFACTS: "${ARTIFACTS:=./artifacts}"

glide install

go vet $(glide novendor)
go test -race -v $(glide novendor)

CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o ${ARTIFACTS}/binaries/linux/amd64/caddy
CGO_ENABLED=0 GOARCH=arm GOARM=6 GOOS=linux go build -o ${ARTIFACTS}/binaries/linux/arm32v6/caddy
CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -o ${ARTIFACTS}/binaries/windows/amd64/caddy.exe

CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags "-X main.DisableTelemetryFlag=true" -o ${ARTIFACTS}/binaries/linux/amd64/caddy-no-telemetry
CGO_ENABLED=0 GOARCH=arm GOARM=6 GOOS=linux go build -ldflags "-X main.DisableTelemetryFlag=true" -o ${ARTIFACTS}/binaries/linux/arm32v6/caddy-no-telemetry
CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -ldflags "-X main.DisableTelemetryFlag=true" -o ${ARTIFACTS}/binaries/windows/amd64/caddy-no-telemetry.exe
