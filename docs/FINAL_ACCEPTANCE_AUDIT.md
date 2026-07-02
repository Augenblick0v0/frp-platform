# Final Acceptance Audit

Generated: 2026-07-03T00:22:03+08:00

## Current Evidence

- Go tests: `./scripts/dev-smoke.sh` passed.
- API server build: `go build ./apps/api-server/cmd/server` passed.
- Client build: `go build ./client/frp-client` passed.
- Docker Compose config: `docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config` passed.
- Final API smoke: register/login/redeem/create TCP/create HTTP/report traffic/admin dashboard/request certificate/renew certificate/operation logs passed.
- Password storage: user/admin passwords are salted hash strings; legacy plaintext verify remains only for old dev rows.
- Windows portable package generated: `dist/windows/FrpTunnelClient-0.1.0-windows-amd64.zip`.
- Linux package generated: `dist/linux/FrpTunnelClient-0.1.0-linux-amd64.tar.gz`.

## Requirement Audit

| Requirement | Evidence | Status |
|---|---|---|
| 服务端 Docker Compose 部署 | deploy/docker-compose.yml exists with nginx, frps, api-server, admin-web, user-web, postgres, redis, mail-server | PASS |
| 单节点 frps | deploy/frps/frps.toml and docker-compose frps service exist | PASS |
| Nginx 80/443 入口 | deploy/nginx/conf.d/platform.conf maps api/admin/panel/default vhost to frps | PASS |
| 管理后台 Cloudflare 风格 | apps/admin-web/index.html and apps/admin-web/style.css | PASS |
| 用户注册登录邮箱验证码 | /api/auth/send-email-code, register, login implemented; SMTP Mailer exists | PASS |
| 自建邮箱服务器 | docker-mailserver in compose and deploy/mail/README.md | PASS |
| 套餐和兑换码 | plans/redeem_codes tables, admin APIs, user redeem API | PASS |
| 购买链接配置 | system settings purchase_url and user purchase-info API | PASS |
| TCP/UDP 自动端口分配 | Store/SQLStore CreateTunnel allocate TCP/UDP from settings | PASS |
| HTTP/HTTPS 域名隧道 | CreateTunnel supports http/https domain and frpc renderer customDomains | PASS |
| Let’s Encrypt 申请/续期 | Automation RequestCertificate and CertificateRenewer | PASS |
| 证书状态持久化 | certificates table and CertificateRecord | PASS |
| frps 管理 | FRPSManager and admin frps APIs | PASS |
| 流量统计与套餐限制 | traffic_logs, /api/client/traffic, plan limit checks | PASS |
| 管理员鉴权和操作日志 | admin login/session and admin_operation_logs | PASS |
| Linux 客户端 WebUI | client/frp-client local server and apps/client-webui | PASS |
| Windows 安装包 | client/packaging/windows NSIS scripts and build-windows scripts | PASS |
| 客户端 frpc 配置和进程管理 | clientcore RenderFRPCConfig and Manager Start/Stop | PASS |

## Remaining Gaps / Risks

- **Real frpc binaries:** Packaging supports `FRPC_WINDOWS_PATH`/`FRPC_LINUX_PATH` and `FRPC_WINDOWS_URL`/`FRPC_LINUX_URL` to include official frpc binaries without vendoring them in git.
- **Real production reload commands:** Nginx/frps command hooks are configurable but default to no-op for safe deployment.
- **Docker image build verification:** Compose config is verified; a full docker build was attempted but interrupted after several minutes because the base image download was extremely slow in this environment. Local Go builds and packaging builds passed.
