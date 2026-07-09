# FRP Tunnel Platform Release 0.1.4

Release date: 2026-07-09

## Artifacts

```text
./windows/FrpTunnelClient-0.1.4-windows-amd64.zip
./windows/installer.nsi
./linux/FrpTunnelClient-0.1.4-linux-amd64.tar.gz
./fnos/api-server
./SHA256SUMS-0.1.4.txt
./RELEASE-0.1.4.md
```

## Verification

```bash
sha256sum -c SHA256SUMS-0.1.4.txt
```

## Notes

- Includes security remediation for authentication defaults, tunnel lifecycle checks, and local client API access.
- Windows and Linux client packages now include the built local WebUI assets instead of development dependencies.
- fnOS deployment uses `dist/fnos/api-server` with `deploy/api-server-local.Dockerfile`.
- Windows setup exe is generated only when `makensis` is installed; this build produced the portable ZIP and NSIS script.
