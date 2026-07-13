#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-${VERSION:-0.1.5}}"
cd "$ROOT/dist"
if [[ ! -f "SHA256SUMS-${VERSION}.txt" ]]; then
  echo "missing dist/SHA256SUMS-${VERSION}.txt" >&2
  exit 1
fi
sha256sum -c "SHA256SUMS-${VERSION}.txt"
cd "$ROOT"
./scripts/check-no-secrets.sh
