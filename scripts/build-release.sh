#!/usr/bin/env bash
# build-release.sh <tag> — cross-compile pi-run for release.
#
# Produces bin/pi-run-<os>-<arch> for linux/darwin/windows × amd64/arm64,
# stamped with <tag> via -ldflags.
set -euo pipefail

if [ $# -ne 1 ]; then
  echo "usage: $0 <tag>" >&2
  exit 2
fi
TAG="$1"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LDFLAGS="-X github.com/forrestthomas1/pi-harness/internal/cli.Version=${TAG}"

cd "$ROOT"
mkdir -p bin

for GOOS in linux darwin windows; do
  for GOARCH in amd64 arm64; do
    EXT=""
    if [ "$GOOS" = "windows" ]; then EXT=".exe"; fi
    OUT="bin/pi-run-${GOOS}-${GOARCH}${EXT}"
    echo "building ${OUT} ..."
    GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o "$OUT" ./cmd/pi-run
  done
done

echo "done: $(ls bin/pi-run-* | tr '\n' ' ')"
