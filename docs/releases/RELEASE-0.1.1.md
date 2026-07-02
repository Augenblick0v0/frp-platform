# FRP Tunnel Platform Release 0.1.1

## Artifacts

```text
./RELEASE-0.1.1.md
./SHA256SUMS-0.1.1.txt
./linux/FrpTunnelClient-0.1.1-linux-amd64.tar.gz
./linux/frp-client/README-FRPC.txt
./linux/frp-client/frp-client
./linux/frp-client/frp-tunnel-client.service
./linux/frp-client/install.sh
./windows/FrpTunnelClient-0.1.1-windows-amd64.zip
./windows/FrpTunnelClient/README-FRPC.txt
./windows/FrpTunnelClient/frp-client.exe
./windows/FrpTunnelClient/open-webui.bat
./windows/FrpTunnelClient/start-client.bat
./windows/installer.nsi
```

## Verification

```bash
sha256sum -c SHA256SUMS-0.1.1.txt
```

## Notes

- Windows setup exe is generated only when `makensis` is installed.
- If official frpc binaries are not provided, package contains README-FRPC.txt with download instructions.
