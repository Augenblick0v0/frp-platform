#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
DIST="$ROOT/dist/linux"
APP="$DIST/frp-client"
VERSION="${VERSION:-0.1.4}"
FRPC_LINUX_PATH="${FRPC_LINUX_PATH:-}"
FRPC_LINUX_URL="${FRPC_LINUX_URL:-}"

rm -rf "$DIST"
mkdir -p "$APP/webui" "$APP/config" "$APP/logs"

(
  cd "$ROOT/apps/client-webui"
  npm install
  npm run build
)

(
  cd "$ROOT/client/frp-client"
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o "$APP/frp-client" .
)

cp -a "$ROOT/apps/client-webui/dist/." "$APP/webui/"
cp "$ROOT/client/packaging/linux/frp-tunnel-client.service" "$APP/frp-tunnel-client.service"
cp "$ROOT/client/packaging/linux/install.sh" "$APP/install.sh"
cat > "$APP/config/client.example.json" <<JSON
{
  "api_base": "https://api.example.com",
  "local_webui": "http://127.0.0.1:18080",
  "frpc_path": "/opt/frp-client/frpc"
}
JSON

if [[ -n "$FRPC_LINUX_PATH" && -f "$FRPC_LINUX_PATH" ]]; then
  cp "$FRPC_LINUX_PATH" "$APP/frpc"
elif [[ -n "$FRPC_LINUX_URL" ]]; then
  TMPGZ="$(mktemp -t frpc-linux-XXXX.tar.gz)"
  curl -LfsS "$FRPC_LINUX_URL" -o "$TMPGZ"
  tar -xOzf "$TMPGZ" --wildcards '*/frpc' > "$APP/frpc"
  rm -f "$TMPGZ"
else
  cat > "$APP/README-FRPC.txt" <<'TXT'
Please place the Linux amd64 frpc binary at /opt/frp-client/frpc, or build with FRPC_LINUX_URL pointing to an frp release tar.gz.
TXT
fi

chmod +x "$APP/frp-client" "$APP/install.sh"
if [[ -f "$APP/frpc" ]]; then chmod +x "$APP/frpc"; fi
(cd "$DIST" && tar -czf "FrpTunnelClient-${VERSION}-linux-amd64.tar.gz" frp-client)
echo "$DIST"
