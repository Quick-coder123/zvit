#!/bin/sh
# Builds dist/potik3d.exe for Windows with the version stamped in.
set -e
cd "$(dirname "$0")"
VERSION=${VERSION:-$(date +%Y.%m.%d-%H%M)}
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath \
  -ldflags "-s -w -H windowsgui -X potik3d/internal/app.Version=$VERSION" \
  -o dist/potik3d.exe ./cmd/potik3d
echo "dist/potik3d.exe ($VERSION)"
