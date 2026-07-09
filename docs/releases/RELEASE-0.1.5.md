# FRP Tunnel Platform Release 0.1.5

Release date: 2026-07-09

## Artifacts

```text
./windows/FrpTunnelClient-0.1.5-windows-amd64.zip
./windows/installer.nsi
./linux/FrpTunnelClient-0.1.5-linux-amd64.tar.gz
./fnos/api-server
./SHA256SUMS-0.1.5.txt
./RELEASE-0.1.5.md
```

## Verification

```bash
sha256sum -c SHA256SUMS-0.1.5.txt
```

## Notes

- Promotes the ME Frp / frp-panel React + Vite layout line to `master`.
- Keeps the security hardening for non-default production secrets, random auth tokens, random email codes, local client API token protection, and tunnel lifecycle checks.
- Updates client download links to `0.1.5` through the user topology API.
- fnOS deployment uses `dist/fnos/api-server` with `deploy/api-server-local.Dockerfile`.
- Windows setup exe is generated only when `makensis` is installed; this build produced the portable ZIP and NSIS script.
