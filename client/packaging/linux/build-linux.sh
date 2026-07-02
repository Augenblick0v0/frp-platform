#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
DIST="$ROOT/dist/linux"
APP="$DIST/frp-client"
VERSION="${VERSION:-0.1.0}"
FRPC_LINUX_PATH="${FRPC_LINUX_PATH:-}"
rm -rf "$DIST"
mkdir -p "$APP/webui" "$APP/config" "$APP/logs"
cd "$ROOT/client/frp-client"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o "$APP/frp-client" .
cp -a "$ROOT/apps/client-webui/." "$APP/webui/"
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
else
  cat > "$APP/README-FRPC.txt" <<'TXT'
请把 Linux amd64 版 frpc 放到 /opt/frp-client/frpc 或当前目录后再执行 install.sh。
下载地址：https://github.com/fatedier/frp/releases
TXT
fi
chmod +x "$APP/frp-client" "$APP/install.sh"
if [[ -f "$APP/frpc" ]]; then chmod +x "$APP/frpc"; fi
(cd "$DIST" && tar -czf "FrpTunnelClient-${VERSION}-linux-amd64.tar.gz" frp-client)
echo "$DIST"
