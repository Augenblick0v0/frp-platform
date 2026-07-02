# FRP 平台架构设计

## 1. 架构原则

- 单节点优先，后续预留多节点扩展。
- 服务端是唯一可信数据源。
- 客户端可以发起隧道创建，但必须由服务端校验和保存。
- Nginx 统一承载 80/443，不让用户 HTTP/HTTPS 隧道带端口访问。
- TCP/UDP 使用端口池自动分配。
- HTTP/HTTPS 使用域名分发。
- HTTPS 由平台统一申请和管理证书。

## 2. 服务拓扑

```text
Internet
  ↓
Nginx
  ├── admin.example.com → admin-web
  ├── panel.example.com → user-web
  ├── api.example.com   → api-server
  └── custom domains    → frps vhost HTTP port

api-server
  ├── PostgreSQL
  ├── Redis
  ├── frps admin/control integration
  ├── cert-manager
  ├── mail-server SMTP
  └── nginx config manager

client
  ├── local webui
  ├── embedded frpc
  └── api-server
```

## 3. 推荐技术栈

- Backend：Go + Gin/Fiber。
- Frontend：React + Vite + Ant Design。
- Client Core：Go。
- Database：PostgreSQL。
- Cache：Redis。
- Reverse Proxy：Nginx。
- FRP：官方 frps/frpc。
- Mail：docker-mailserver 或 Postfix 组合。
- Certificate：Certbot/ACME client。
- Packaging：Windows NSIS/Wix，Linux tar.gz + systemd。

## 3.1 前端视觉规范

前端统一采用 Cloudflare 网页风格，覆盖 admin-web、user-web、client local webui。

要求：

- 使用浅色 SaaS 控制台风格。
- 主色使用 Cloudflare 风格橙色。
- 页面结构以顶部导航、侧边栏、卡片、表格、详情抽屉为主。
- 状态使用清晰 badge，例如 active、expired、disabled、error。
- 仪表盘使用简洁数据卡片和趋势图。
- 表单页保持短路径操作，避免复杂弹窗嵌套。

## 4. 请求路径

### 4.1 管理后台

```text
admin.example.com
  → Nginx
  → admin-web
  → api-server
```

### 4.2 用户 API

```text
client / user-web
  → api.example.com
  → Nginx
  → api-server
```

### 4.3 HTTP 隧道

```text
http://app.user.com
  → Nginx :80
  → proxy_set_header Host app.user.com
  → frps vhost HTTP port
  → frpc
  → user local service
```

### 4.4 HTTPS 隧道

```text
https://app.user.com
  → Nginx :443 with certificate
  → TLS termination
  → proxy_set_header Host app.user.com
  → frps vhost HTTP port
  → frpc
  → user local service
```

### 4.5 TCP/UDP 隧道

```text
external server port
  → frps
  → frpc
  → user local service
```

## 5. 配置生成

### 5.1 frps 配置

由部署阶段生成基础 frps 配置，包括：

- bindPort。
- token。
- vhostHTTPPort。
- dashboard 可选。
- log 配置。

### 5.2 frpc 配置

客户端不让用户直接编辑 frpc 配置。服务端根据隧道数据生成配置，客户端拉取后写入本地临时配置并启动 frpc。

### 5.3 Nginx 配置

系统根据域名生成 Nginx server block：

- HTTP 默认反代。
- HTTPS 证书配置。
- ACME challenge 路径。

生成后执行：

```bash
nginx -t && nginx -s reload
```

## 6. 状态流转

### 6.1 隧道状态

```text
created
pending_domain_check
pending_certificate
active
stopped
disabled
error
```

### 6.2 域名状态

```text
pending
valid
invalid
disabled
```

### 6.3 证书状态

```text
none
requesting
issued
failed
renewing
expired
```

## 7. 安全边界

- 用户只能管理自己的隧道。
- 套餐权限由服务端校验。
- 客户端提交的端口、域名、协议都不能直接信任。
- 远程端口必须由服务端分配。
- 域名必须唯一绑定。
- 管理员操作要记录日志。
