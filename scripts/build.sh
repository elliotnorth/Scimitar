#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p dist
for arch in amd64 arm64; do
	CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -ldflags='-s -w' -o "dist/scimitar-linux-$arch" ./cmd/scimitar
done
ls -lh dist
