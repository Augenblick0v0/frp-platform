#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
DIST="$ROOT/dist/windows"
APP="$DIST/FrpTunnelClient"
VERSION="${VERSION:-0.1.0}"
FRPC_WINDOWS_PATH="${FRPC_WINDOWS_PATH:-}"
FRPC_WINDOWS_URL="${FRPC_WINDOWS_URL:-}"
rm -rf "$DIST"
mkdir -p "$APP/webui" "$APP/config" "$APP/logs"
cd "$ROOT/client/frp-client"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o "$APP/frp-client.exe" .
cp -a "$ROOT/apps/client-webui/." "$APP/webui/"
cat > "$APP/config/client.example.json" <<JSON
{
  "api_base": "https://api.example.com",
  "local_webui": "http://127.0.0.1:18080",
  "frpc_path": "frpc.exe"
}
JSON
if [[ -n "$FRPC_WINDOWS_PATH" && -f "$FRPC_WINDOWS_PATH" ]]; then
  cp "$FRPC_WINDOWS_PATH" "$APP/frpc.exe"
elif [[ -n "$FRPC_WINDOWS_URL" ]]; then
  TMPZIP="$(mktemp -t frpc-windows-XXXX.zip)"
  curl -LfsS "$FRPC_WINDOWS_URL" -o "$TMPZIP"
  unzip -p "$TMPZIP" '*/frpc.exe' > "$APP/frpc.exe"
  rm -f "$TMPZIP"
else
  cat > "$APP/README-FRPC.txt" <<'TXT'
请把 Windows 版 frpc.exe 放到本目录，或构建时设置 FRPC_WINDOWS_URL 自动下载官方 release zip。
下载地址：https://github.com/fatedier/frp/releases
安装包脚本会把本目录整体安装到 Program Files。
TXT
fi
cat > "$APP/start-client.bat" <<'BAT'
@echo off
cd /d "%~dp0"
frp-client.exe -addr 127.0.0.1:18080 -web webui -workdir "%LOCALAPPDATA%\FrpTunnelClient" -frpc "%~dp0frpc.exe"
BAT
cat > "$APP/open-webui.bat" <<'BAT'
@echo off
start http://127.0.0.1:18080
BAT
cp "$ROOT/client/packaging/windows/installer.nsi" "$DIST/installer.nsi"
if command -v zip >/dev/null 2>&1; then
  (cd "$DIST" && zip -qr "FrpTunnelClient-${VERSION}-windows-amd64.zip" FrpTunnelClient installer.nsi)
fi
if command -v makensis >/dev/null 2>&1; then
  makensis -DVERSION="$VERSION" -DAPPDIR="$APP" -DOUTFILE="$DIST/FrpTunnelClient-${VERSION}-setup.exe" "$DIST/installer.nsi"
else
  echo "makensis not found; generated portable zip and NSIS script only" >&2
fi
echo "$DIST"
