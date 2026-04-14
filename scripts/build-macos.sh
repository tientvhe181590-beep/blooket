#!/usr/bin/env bash
# Run on macOS with Xcode CLI tools (clang). Produces a native binary in the repo root.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
export CGO_ENABLED=1

arch="${1:-$(uname -m)}"
out="blooket-csv-exporter"
case "$arch" in
  arm64|aarch64) GOARCH=arm64 ;;
  amd64|x86_64)  GOARCH=amd64 ;;
  *) echo "usage: $0 [arm64|amd64]"; exit 1 ;;
esac

echo "Building GOOS=darwin GOARCH=${GOARCH} ..."
# Must run on a Mac (same Apple SDK as target). Cross-compiling Fyne darwin from Windows is not supported.
time env GOOS=darwin "GOARCH=${GOARCH}" CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o "${out}-darwin-${GOARCH}" .
