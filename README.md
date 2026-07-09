# FRP Tunnel Platform

FRP Tunnel Platform 是一套可私有部署、可商业化运营的内网穿透平台。系统按 frp-panel 的角色语言组织为 Master / Server(FRPS) / Client(FRPC) / Visitor，并将后台管理端、用户控制台、本地客户端 UI 拆成三个独立前端。

## 当前入口

- 用户控制台：`http://192.168.110.56:18188`
- 后台管理端：`http://192.168.110.56:18189`
- 本地客户端 WebUI：默认 `http://127.0.0.1:18080`

## 目录结构

```text
apps/api-server      Master Control Plane，Go API 服务
apps/admin-web       Admin Console，React + Vite + Ant Design
apps/user-web        User Console，React + Vite + Ant Design
apps/client-webui    Client(FRPC) 本地 WebUI，React + Vite + Ant Design
apps/node-agent      Server(FRPS) 节点管理代理 Dockerfile
apps/shared/frontend 三端共享前端主题、布局、API client 和组件
client/frp-client    本地 Win/Linux 客户端，控制 frpc 与本地测速服务
deploy               Docker Compose、Nginx、frps 部署模板
docs                 PRD、架构、ADR、验收文档
```

## 架构角色

| 角色 | 实现 | 说明 |
| --- | --- | --- |
| Master Control Plane | `apps/api-server` | 用户、套餐、订单、支付、兑换码、隧道、节点、证书、流量、frpc 配置生成、API 托管测速 |
| Admin Console | `apps/admin-web` | 后台管理端，管理 Master 资源和 Server(FRPS) 节点运维 |
| User Console | `apps/user-web` | 用户控制台，套餐支付、隧道、节点安全字段、客户端、域名证书、测速 |
| Server(FRPS) Node Plane | `frps` + `node-agent` + `node-nginx` | 承载公网入口、frps 连接、HTTP/HTTPS vhost、节点操作 |
| Client(FRPC) | `client/frp-client` + `apps/client-webui` | 用户本地客户端，拉取配置、控制 frpc、提供日志和 benchmark 服务 |
| Visitor | 外部访问者 | 访问 Server(FRPS) 暴露的公网入口 |

## 本地开发

### 前端

```bash
cd apps/user-web && npm install && npm run build
cd apps/admin-web && npm install && npm run build
cd apps/client-webui && npm install && npm run build
```

### API

```bash
cd apps/api-server
go test ./...
go run ./cmd/server
```

如配置 `DATABASE_URL`，API Server 使用 PostgreSQL 并自动执行迁移；否则使用内存存储。

## 飞牛部署

```bash
cd /root/frp-platform
docker compose -f deploy/docker-compose.fnos.yml --env-file deploy/.env.fnos up -d --build api-server user-portal admin-portal
```

`apps/user-web/Dockerfile` 和 `apps/admin-web/Dockerfile` 会先用 Node 构建 React 产物，再由 Nginx 提供静态资源。用户端仍将 `dist/` 下客户端发布包挂到 `/downloads`。

## 关键功能

- mefrp 风格的信息架构：左侧分组菜单、顶部工具栏、工作台指标卡、表格 + Drawer、步骤化创建。
- 后台用户管理可修改用户状态和套餐。
- 套餐管理支持新增与编辑。
- 兑换码生成必须选择套餐。
- 支付方式绑定入口展示微信支付/支付宝与 pay_type/channel 映射。
- 易支付 `wxpay` / `alipay` 下单，兼容微信支付别名和 `wxpay_zg` 通道名。
- FRPS 节点状态、配置、日志、重启、reload、nginx test、nginx reload 操作统一展示在 NodeOperationPanel。
- 用户 topology 只返回安全节点字段，不暴露 node-agent token、支付密钥或 frps token。
- API Server 托管测速：本地 Client(FRPC) 准备 benchmark，Master 创建临时隧道，API Server 发起测速并清理。

## 文档

- 架构映射：`docs/architecture/frp-panel-role-map.md`
- ADR：`docs/adr/0002-frp-panel-role-architecture.md`
- 旧架构文档：`docs/plans/02-ARCHITECTURE.md`
- 分离部署：`deploy/SPLIT_DEPLOYMENT.md`
