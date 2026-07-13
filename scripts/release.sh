#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-${VERSION:-0.1.5}}"
cd "$ROOT"

echo "==> Running tests"
./scripts/dev-smoke.sh

echo "==> Checking tracked files and release roots for secrets"
./scripts/check-no-secrets.sh

echo "==> Building server and client"
mkdir -p dist/fnos
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o dist/fnos/api-server ./apps/api-server/cmd/server
go build ./client/frp-client
rm -f frp-client

echo "==> Validating docker compose"
docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config >/tmp/frp-platform-compose-release.yml

echo "==> Building client packages"
VERSION="$VERSION" ./client/packaging/windows/build-windows.sh >/tmp/frp-platform-windows-release.log
VERSION="$VERSION" ./client/packaging/linux/build-linux.sh >/tmp/frp-platform-linux-release.log

echo "==> Writing checksums"
mkdir -p dist
(
  cd dist
  find . -type f \( -name '*.zip' -o -name '*.tar.gz' -o -name '*.exe' -o -name 'installer.nsi' -o -path './fnos/api-server' \) -print0 | sort -z | xargs -0 sha256sum
) > "dist/SHA256SUMS-${VERSION}.txt"

./scripts/check-no-secrets.sh

cat > "dist/RELEASE-${VERSION}.md" <<MD
# FRP Tunnel Platform Release ${VERSION}

## Artifacts

\`\`\`text
./fnos/api-server
./linux/FrpTunnelClient-${VERSION}-linux-amd64.tar.gz
./nhk-lite-node-deploy.tar.gz
./windows/FrpTunnelClient-${VERSION}-windows-amd64.zip
./windows/installer.nsi
./SHA256SUMS-${VERSION}.txt
\`\`\`

## Checksum Verification

\`\`\`bash
sha256sum -c SHA256SUMS-${VERSION}.txt
\`\`\`

## Highlights

- Added Narwhal Cloud NAT node type for automatic TCP/UDP port-forward allocation on new tunnels.
- Improved admin/user node flows so NAT nodes are clearly labeled and explain their forwarding behavior.
- Hardened API request limits, timeouts, method checks, node binding, and production traffic reporting.
- Serialized SQL tunnel quota allocation to prevent concurrent over-allocation.
- Protected local status and log APIs and switched sensitive client files to atomic private writes.
- Moved browser session tokens out of persistent local storage and added request timeout and error states.
- Added release secret scanning, pinned deployment images, and private runtime token persistence.

## Verification

- Go tests, race detector, and vet passed.
- User, admin, and client web production builds passed.
- Standard, control, node, and fnOS Compose configurations rendered successfully.
- Release checksums and secret-file gate passed.

## Packaging Notes

- Windows setup exe is generated only when \`makensis\` is installed.
- If official frpc binaries are not provided, the package contains README-FRPC.txt with download instructions.
MD

echo "Release artifacts written to dist/"
