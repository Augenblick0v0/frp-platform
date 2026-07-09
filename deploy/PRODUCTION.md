# FRP Tunnel Platform 生产部署手册

本文档用于把当前单节点 FRP 商业化平台部署到 Linux Docker 主机。

## 1. 推荐服务器配置

最低测试：

```text
2 核 CPU
2GB RAM
20GB SSD
```

推荐正式小规模：

```text
2-4 核 CPU
4GB RAM
40GB+ SSD
```

如果启用自建邮件服务器，建议至少 4GB RAM。

## 2. DNS 规划

假设主域名为：

```text
example.com
```

需要配置：

```text
A     api.example.com      <server-ip>
A     admin.example.com    <server-ip>
A     panel.example.com    <server-ip>
A     frp.example.com      <server-ip>
A     mail.example.com     <server-ip>
MX    example.com          mail.example.com
TXT   example.com          v=spf1 mx ~all
TXT   _dmarc.example.com   v=DMARC1; p=quarantine; rua=mailto:postmaster@example.com
```

用户自定义域名需要 CNAME 到：

```text
frp.example.com
```

示例：

```text
app.user.com CNAME frp.example.com
```

## 3. 端口要求

服务器防火墙需要开放：

```text
80/tcp       HTTP / ACME / 用户 HTTP 隧道
443/tcp      HTTPS 用户域名
25/tcp       SMTP 入站
587/tcp      SMTP Submission
993/tcp      IMAP 可选
7000/tcp     frps control
20000-29999/tcp  TCP 隧道端口池
30000-39999/udp  UDP 隧道端口池
```

端口池可在 `.env` 和后台设置中调整。

## 4. 首次部署

```bash
cd deploy
cp .env.example .env
vim .env
```

重点修改：

```env
PLATFORM_DOMAIN=example.com
ADMIN_DOMAIN=admin.example.com
PANEL_DOMAIN=panel.example.com
API_DOMAIN=api.example.com
FRP_ENTRY_DOMAIN=frp.example.com
POSTGRES_PASSWORD=<strong-password>
REDIS_PASSWORD=<strong-password>
FRP_TOKEN=<strong-token>
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=<strong-admin-password>
LETSENCRYPT_EMAIL=admin@example.com
SMTP_USERNAME=noreply@example.com
SMTP_PASSWORD=<mail-password>
SMTP_FROM_EMAIL=noreply@example.com
CERTBOT_DRY_RUN=false
```

启动：

```bash
docker compose up -d
```

检查：

```bash
docker compose ps
curl http://api.example.com/health
```

## 5. 邮件服务器初始化

创建发信账号：

```bash
docker exec -it frp-platform-mail setup email add noreply@example.com '<mail-password>'
docker exec -it frp-platform-mail setup config dkim
```

查看 DKIM：

```bash
cat deploy/mail/config/opendkim/keys/example.com/mail.txt
```

把 DKIM TXT 记录添加到 DNS。

后台测试邮件：

```text
管理后台 → 邮件测试 → 发送测试邮件
```

## 6. Nginx / 证书

默认由 Nginx 监听 80/443。

用户 HTTPS 域名流程：

```text
用户 CNAME 到 frp.example.com
后台检测 CNAME
后台申请证书
后台生成 HTTPS Nginx 配置
后台执行 Nginx reload
用户访问 https://app.user.com
```

`.env` 关键项：

```env
CERTBOT_DRY_RUN=false
CERT_RENEW_BEFORE_DAYS=30
CERT_RENEW_INTERVAL=24h
```

生产环境中建议配置实际 reload 命令：

```env
NGINX_TEST_CMD=nginx -t
NGINX_RELOAD_CMD=nginx -s reload
```

如果 API Server 与 Nginx 不在同一容器命名空间，建议使用受限 sidecar 或宿主机脚本完成 reload。

## 7. frps 管理

默认 API Server 只读查看 frps 配置和日志。

如需 restart/reload，配置：

```env
FRPS_STATUS_CMD=<your-status-command>
FRPS_RESTART_CMD=<your-restart-command>
FRPS_RELOAD_CMD=<your-reload-command>
```

推荐生产做法：

- 通过受限运维脚本控制 frps
- 不直接把 Docker socket 暴露给 API Server
- 所有 restart/reload 操作会记录管理员操作日志

## 8. 客户端发布

生成发布包：

```bash
./scripts/release.sh 0.1.1
```

Windows：

```text
dist/windows/FrpTunnelClient-0.1.1-windows-amd64.zip
```

如果安装了 NSIS，会额外生成：

```text
dist/windows/FrpTunnelClient-0.1.1-setup.exe
```

Linux：

```text
dist/linux/FrpTunnelClient-0.1.1-linux-amd64.tar.gz
```

## 9. 上线验收

### 服务端

```bash
curl http://api.example.com/health
```

后台检查：

```text
管理员登录
创建套餐
生成兑换码
配置购买链接
查看 frps 状态
查看 Nginx 状态
测试邮件
```

### 用户流程

```text
邮箱注册
邮箱验证码
登录
兑换套餐
创建 TCP 隧道
创建 UDP 隧道
创建 HTTP 域名隧道
创建 HTTPS 域名隧道
客户端同步配置
frpc 启动
查看流量
```

### 域名 HTTPS

```bash
curl -I http://app.user.com
curl -I https://app.user.com
```

访问时不应带端口号。

## 10. 备份

建议定期备份：

```text
deploy/postgres/data
deploy/certbot/letsencrypt
deploy/mail/data
deploy/mail/config
deploy/.env
```

PostgreSQL 逻辑备份：

```bash
docker exec frp-platform-postgres pg_dump -U frp_platform frp_platform > backup.sql
```

## 11. 安全建议

- 修改默认管理员密码。
- 使用强 FRP token。
- 生产环境关闭 `CERTBOT_DRY_RUN`。
- 不把 Docker socket 直接暴露给 API Server。
- 定期备份数据库和证书。
- 将后台域名放到可信网络或加额外访问控制。

## 控制面与 frps 节点分离部署

如果不希望后台和 frps 节点运行在同一台服务器，使用：

```text
deploy/docker-compose.control.yml  控制面：后台、用户面板、API、数据库、邮件服务
deploy/docker-compose.node.yml     节点面：frps、节点 Nginx、node-agent、证书运行环境
```

完整步骤见：`deploy/SPLIT_DEPLOYMENT.md`。

分离模式下，在控制面 `.env.control` 配置：

```env
NODE_AGENT_URL=http://NODE_SERVER_IP:8090
NODE_AGENT_TOKEN=<same-token-as-node>
```

在节点面 `.env.node` 配置相同的：

```env
NODE_AGENT_TOKEN=<same-token-as-control>
```

配置后，后台的 frps 管理、Nginx reload、HTTPS 配置生成、Let’s Encrypt 申请和续期会通过 `node-agent` 在节点服务器执行。



## 2026-07-09 ??????

?????????

```env
ADMIN_PASSWORD=<non-default-strong-password>
FRP_TOKEN=<random-32-byte-or-longer-token>
NODE_AGENT_TOKEN=<random-node-agent-token>
```

???????`admin123456` ? `change-me`?API Server ???????? `ADMIN_PASSWORD`/`FRP_TOKEN`?`/api/client/tunnels` ?????? token ? `change-me`?

????? WebUI ?? `/api/local-token` ???? token??????????? API ?????

```http
X-Local-Token: <local-token>
```

node-agent ??????? `127.0.0.1:8090:8090`?????????????????? IP ?????? `NODE_AGENT_TOKEN`?

frps ???? `deploy/frps/frps.toml` ???? `__REPLACE_WITH_FRP_TOKEN__` ???????? compose ??? `FRP_TOKEN`?
