# FRP Tunnel Platform

可私有部署、可商业化运营的 FRP 内网穿透平台。当前仓库按 `/root/frp-platform-plans` 的规划落地，采用单节点服务端、Nginx 80/443 入口、frps、Go API、Cloudflare 风格 WebUI、Linux/Windows 客户端 WebUI。

## 目录

```text
apps/api-server      Go 后端 API
apps/admin-web       管理后台静态前端
apps/user-web        用户面板静态前端
apps/client-webui    客户端内置 WebUI 静态前端
client/frp-client    Go 客户端骨架
deploy               Docker/Nginx/frps 部署模板
docs/plans           产品规划文档
```

## 本地运行 API

```bash
cd apps/api-server
go test ./...
go run ./cmd/server
```

默认监听 `:8080`。如果设置 `DATABASE_URL`，API Server 会自动使用 PostgreSQL 并执行迁移；否则使用内存存储。

## 静态前端

可直接用浏览器打开：

```text
apps/admin-web/index.html
apps/user-web/index.html
apps/client-webui/index.html
```

## 部署模板

```bash
cd deploy
cp .env.example .env
docker compose up -d
```
