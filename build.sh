#!/usr/bin/env bash
# Same as build.bat but for Git Bash / WSL / macOS / Linux.
# Usage: ./build.sh (interactive; on Windows shells, waits for a key)
#        ./build.sh nopause  (for Make / CI; no wait)

set -euo pipefail
root="$(cd "$(dirname "$0")" && pwd)"
cd "$root"

nopause=false
[[ "${1:-}" == "nopause" ]] && nopause=true

echo ""
echo "========================================"
echo " Blooket CSV Exporter - build"
echo "========================================"
echo ""

export CGO_ENABLED=1
out="blooket-csv-exporter"
ldflags="-s -w"

# Windows-ish (Git Bash / MSYS): GUI subsystem + .exe
if [[ "${OSTYPE:-}" == msys* || "${OSTYPE:-}" == cygwin* ]]; then
  out="${out}.exe"
  ldflags="-H windowsgui ${ldflags}"
fi

echo "[1/2] CGO_ENABLED=1 (required for Fyne)"
echo "[2/2] go build ..."
echo ""

start=$(date +%s)
go build -trimpath -ldflags="${ldflags}" -o "${out}" .
end=$(date +%s)
elapsed=$((end - start))

echo ""
echo "*** BUILD OK *** (${elapsed}s)"
echo "Output: ${root}/${out}"
echo ""

if ! $nopause; then
  if [[ -n "${WINDIR:-}" || "${OSTYPE:-}" == msys* || "${OSTYPE:-}" == cygwin* ]]; then
    # shellcheck disable=SC2162
    read -n 1 -s -r -p "Press any key to close..."
    echo ""
  fi
fi
