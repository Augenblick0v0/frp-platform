# FRP Tunnel Platform

可私有部署、可商业化运营的 FRP 内网穿透平台。当前仓库按 `/root/frp-platform-plans` 的规划落地，支持单节点一体化部署，也支持控制面与 frps 节点分离部署；包含 Nginx 80/443 入口、frps、Go API、Cloudflare 风格 WebUI、Linux/Windows 客户端 WebUI。

## 目录

```text
apps/api-server      Go 后端 API 与 node-agent 节点代理
apps/node-agent      节点代理 Dockerfile
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

单机一体化部署：

```bash
cd deploy
cp .env.example .env
docker compose up -d --build
```

控制面与 frps 节点分离部署：

```bash
# 控制面服务器
cd deploy
cp .env.control.example .env.control
docker compose --env-file .env.control -f docker-compose.control.yml up -d --build

# 节点服务器
cd deploy
cp .env.node.example .env.node
docker compose --env-file .env.node -f docker-compose.node.yml up -d --build
```


## 生产部署与发布

- 生产部署手册：`deploy/PRODUCTION.md`
- 控制面/节点分离部署：`deploy/SPLIT_DEPLOYMENT.md`
- 发布脚本：`./scripts/release.sh 0.1.1`
- 发布校验：`./scripts/verify-release.sh 0.1.1`
- 最终验收审计：`docs/FINAL_ACCEPTANCE_AUDIT.md`
