#!/bin/sh
# Build the server for the dev box and for the router.
set -eu
cd "$(dirname "$0")/.."
mkdir -p dist
go vet ./...
go build -o dist/showcase-server-local .
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/showcase-server-linux-arm64 .
ls -la dist
