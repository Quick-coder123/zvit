#!/bin/sh
# Builds dist/potik.exe for Windows with the version stamped in.
set -e
cd "$(dirname "$0")"
VERSION=${VERSION:-$(date +%Y.%m.%d-%H%M)}
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath \
  -ldflags "-s -w -H windowsgui -X potik/internal/game.Version=$VERSION" \
  -o dist/potik.exe ./cmd/potik
echo "dist/potik.exe ($VERSION)"
