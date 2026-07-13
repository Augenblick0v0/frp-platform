#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail=0
while IFS= read -r path; do
  case "$path" in
    *.env.example|*.env.*.example)
      ;;
    *.env|*.env.*|*.pem|*.key|*id_rsa*|*id_ed25519*)
      echo "forbidden tracked secret-like file: $path" >&2
      fail=1
      ;;
  esac
done < <(git ls-files)

while IFS= read -r archive; do
  case "$archive" in
    *.zip) entries="$(unzip -Z1 "$archive")" ;;
    *.tar.gz) entries="$(tar -tzf "$archive")" ;;
    *) continue ;;
  esac
  while IFS= read -r entry; do
    base="${entry##*/}"
    case "$base" in
      .env.example|.env.*.example) ;;
      .env|.env.*|*.pem|*.key|id_rsa*|id_ed25519*)
        echo "forbidden secret-like file inside release archive: $archive ($entry)" >&2
        fail=1
        ;;
    esac
  done <<<"$entries"
done < <(find dist -type f \( -name '*.zip' -o -name '*.tar.gz' \) 2>/dev/null || true)

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "secret file gate passed"
