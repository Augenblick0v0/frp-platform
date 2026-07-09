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
cd "$ROOT/client/frp-client"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o "$APP/frp-client" .
WEB_DIST="$ROOT/apps/client-webui/dist"
if [[ -f "$WEB_DIST/index.html" ]]; then
  cp -a "$WEB_DIST/." "$APP/webui/"
else
  cp "$ROOT/apps/client-webui/index.html" "$ROOT/apps/client-webui/style.css" "$APP/webui/"
fi
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
请把 Linux amd64 版 frpc 放到 /opt/frp-client/frpc，或构建时设置 FRPC_LINUX_URL 自动下载官方 release tar.gz。
下载地址：https://github.com/fatedier/frp/releases
TXT
fi
chmod +x "$APP/frp-client" "$APP/install.sh"
if [[ -f "$APP/frpc" ]]; then chmod +x "$APP/frpc"; fi
(cd "$DIST" && tar -czf "FrpTunnelClient-${VERSION}-linux-amd64.tar.gz" frp-client)
echo "$DIST"
