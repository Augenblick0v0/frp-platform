#!/usr/bin/env bash
set -euo pipefail
SRC="$(cd "$(dirname "$0")" && pwd)"
install -d /opt/frp-client /var/lib/frp-client /var/log/frp-client
cp -a "$SRC/frp-client" /opt/frp-client/frp-client
cp -a "$SRC/webui" /opt/frp-client/webui
if [[ -f "$SRC/frpc" ]]; then cp -a "$SRC/frpc" /opt/frp-client/frpc; fi
cp -a "$SRC/frp-tunnel-client.service" /etc/systemd/system/frp-tunnel-client.service
chmod +x /opt/frp-client/frp-client
if [[ -f /opt/frp-client/frpc ]]; then chmod +x /opt/frp-client/frpc; fi
systemctl daemon-reload
systemctl enable frp-tunnel-client.service
echo "Installed. Start with: systemctl start frp-tunnel-client"
echo "WebUI: http://127.0.0.1:18080"
