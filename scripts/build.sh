#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
version="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
rm -rf dist
mkdir -p dist
for arch in amd64 arm64; do
	CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -ldflags="-s -w -X main.version=$version" -o "dist/scimitar-linux-$arch" ./cmd/scimitar
done
cp scripts/install.sh dist/install.sh
(cd dist && sha256sum scimitar-linux-* install.sh >checksums.txt)
ls -lh dist
