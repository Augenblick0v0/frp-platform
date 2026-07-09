#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-${VERSION:-0.1.5}}"
cd "$ROOT"

echo "==> Running tests"
./scripts/dev-smoke.sh

echo "==> Building server and client"
go build ./apps/api-server/cmd/server
go build ./client/frp-client
rm -f server frp-client

echo "==> Validating docker compose"
docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config >/tmp/frp-platform-compose-release.yml

echo "==> Building client packages"
VERSION="$VERSION" ./client/packaging/windows/build-windows.sh >/tmp/frp-platform-windows-release.log
VERSION="$VERSION" ./client/packaging/linux/build-linux.sh >/tmp/frp-platform-linux-release.log

echo "==> Writing checksums"
mkdir -p dist
(
  cd dist
  find . -type f \( -name '*.zip' -o -name '*.tar.gz' -o -name '*.exe' -o -name 'installer.nsi' \) -print0 | sort -z | xargs -0 sha256sum
) > "dist/SHA256SUMS-${VERSION}.txt"

cat > "dist/RELEASE-${VERSION}.md" <<MD
# FRP Tunnel Platform Release ${VERSION}

## Artifacts

\`\`\`text
$(cd dist && find . -maxdepth 3 -type f | sort)
\`\`\`

## Verification

\`\`\`bash
sha256sum -c SHA256SUMS-${VERSION}.txt
\`\`\`

## Notes

- Windows setup exe is generated only when \`makensis\` is installed.
- If official frpc binaries are not provided, package contains README-FRPC.txt with download instructions.
MD

echo "Release artifacts written to dist/"
