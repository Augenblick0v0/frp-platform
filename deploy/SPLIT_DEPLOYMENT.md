# 控制面与 frps 节点分离部署

当前项目支持两种部署方式：

1. `deploy/docker-compose.yml`：单机一体化部署，后台、数据库、Nginx、frps 在同一台服务器。
2. `deploy/docker-compose.control.yml` + `deploy/docker-compose.node.yml`：控制面和节点面分离部署。

分离后的角色：

```text
控制面服务器：
- api-server
- admin-web
- user-web
- postgres
- redis
- mail-server
- control-nginx

节点服务器：
- frps
- node-nginx
- node-agent
- certbot runtime
```

## 通信关系

```text
用户浏览器 -> 控制面 Nginx -> admin-web / user-web / api-server
客户端 frpc -> 节点服务器 frps:7000
外部访问隧道域名 -> 节点服务器 Nginx:80/443 -> frps vhostHTTPPort:8080
控制面 api-server -> 节点服务器 node-agent:8090 -> 节点本地 nginx/frps/certbot
```

`node-agent` 是节点管理代理，负责：

- 写入 HTTPS 域名的 Nginx 配置；
- 调用 certbot 申请 Let’s Encrypt 证书；
- 测试和 reload 节点 Nginx；
- 读取 frps 配置和日志；
- 重启/重载 frps。

## 一、部署节点服务器

在节点服务器上：

```bash
cd deploy
cp .env.node.example .env.node
```

编辑 `.env.node`：

```env
FRP_ENTRY_DOMAIN=frp.example.com
FRP_TOKEN=change-me
NODE_AGENT_TOKEN=change-me-node-agent-token
CERTBOT_DRY_RUN=true
```

首次测试建议保持：

```env
CERTBOT_DRY_RUN=true
```

确认域名和 80/443 解析正确后再改为：

```env
CERTBOT_DRY_RUN=false
```

启动节点：

```bash
docker compose --env-file .env.node -f docker-compose.node.yml up -d --build
```

检查节点代理：

```bash
curl http://127.0.0.1:8090/health
curl -H "Authorization: Bearer $NODE_AGENT_TOKEN" http://127.0.0.1:8090/api/frps/status
```

生产环境建议只允许控制面服务器访问节点服务器的 `8090` 端口。

## 二、部署控制面服务器

在控制面服务器上：

```bash
cd deploy
cp .env.control.example .env.control
```

编辑 `.env.control`：

```env
PLATFORM_DOMAIN=example.com
ADMIN_DOMAIN=admin.example.com
USER_DOMAIN=panel.example.com
API_DOMAIN=api.example.com
FRP_ENTRY_DOMAIN=frp.example.com
SERVER_ADDR=frp.example.com
NODE_AGENT_URL=http://NODE_SERVER_IP:8090
NODE_AGENT_TOKEN=change-me-node-agent-token
```

`NODE_AGENT_TOKEN` 必须和节点服务器 `.env.node` 一致。

启动控制面：

```bash
docker compose --env-file .env.control -f docker-compose.control.yml up -d --build
```

## 三、后台功能在分离模式下的行为

配置 `NODE_AGENT_URL` 后，后台这些功能会自动转发到节点服务器执行：

```text
/admin frps 状态、配置、日志、重启、重载
/admin Nginx 配置生成、测试、reload
/admin Let’s Encrypt 证书申请和续期
```

如果没有配置 `NODE_AGENT_URL`，系统继续使用本地一体化模式。

## 四、DNS 绑定建议

控制面域名：

```text
admin.example.com -> 控制面服务器 IP
panel.example.com -> 控制面服务器 IP
api.example.com   -> 控制面服务器 IP
```

节点入口域名：

```text
frp.example.com -> 节点服务器 IP
*.frp.example.com 或用户自定义 CNAME -> frp.example.com
```

用户自定义域名：

```text
用户域名 CNAME 到 frp.example.com
后台检查 CNAME 后，调用 node-agent 在节点 Nginx 上生成 HTTPS 配置并申请证书
```

## 五、端口开放

控制面服务器：

```text
80/tcp      控制面 Web 入口
25/tcp      SMTP，可按需开放
587/tcp     SMTP submission，可按需开放
993/tcp     IMAP，可按需开放
```

节点服务器：

```text
80/tcp                 HTTP 隧道入口和 ACME 验证
443/tcp                HTTPS 隧道入口
7000/tcp               frpc 连接 frps
20000-29999/tcp        TCP 隧道端口池
30000-39999/udp        UDP 隧道端口池
8090/tcp               node-agent，仅允许控制面访问
```

## 六、4H4G 推荐部署

4H4G 单节点测试时可以一体化部署；如果拆分，推荐：

```text
控制面：2H2G 起步
节点面：2H2G 起步，带宽越高越好
```

如果只有一台 4H4G，也可以使用分离 compose 在同一台机器上运行，便于以后迁移节点。

## 七、多节点管理与节点绑定

后台现在支持多个 frps 节点。推荐流程：

### 1. 在管理后台创建节点

进入管理后台的“多节点管理”，填写：

```text
节点名称：edge-node-1
节点 Agent URL：http://NODE_SERVER_IP:8090
节点入口域名：frp.example.com
frpc 连接地址：frp.example.com
frps 端口：7000
TCP 端口池：20000-29999
UDP 端口池：30000-39999
```

点击“新增节点并生成绑定 Token”。后台会返回：

```text
NODE_BIND_TOKEN=<后台生成的 bind token>
```

### 2. 在节点服务器配置绑定信息

在节点服务器 `.env.node` 中填写：

```env
CONTROL_PLANE_URL=https://api.example.com
NODE_BIND_TOKEN=<后台生成的 bind token>
NODE_NAME=edge-node-1
NODE_PUBLIC_AGENT_URL=http://NODE_SERVER_IP:8090
NODE_PUBLIC_URL=http://NODE_SERVER_IP:8090
FRP_ENTRY_DOMAIN=frp.example.com
SERVER_ADDR=frp.example.com
```

如果 `NODE_AGENT_TOKEN` 留空，节点首次绑定后会从后台领取该节点专属的 agent token，并用于后续远程管理鉴权。

### 3. 启动或重启节点

```bash
docker compose --env-file .env.node -f docker-compose.node.yml up -d --build
```

节点启动后，`node-agent` 会每 60 秒向控制面调用：

```text
POST /api/nodes/bind
```

绑定成功后，后台节点状态变为：

```text
online
```

### 4. 远程管理节点

管理后台可对每个节点执行：

```text
frps 状态
查看 frps 配置
查看 frps 日志
重启 frps
重载 frps
测试节点 Nginx
重载节点 Nginx
```

对应 API：

```text
GET  /api/admin/nodes
POST /api/admin/nodes
GET  /api/admin/nodes/{id}
POST /api/admin/nodes/{id}/status
GET  /api/admin/nodes/{id}/frps-config
GET  /api/admin/nodes/{id}/frps-logs
POST /api/admin/nodes/{id}/frps-restart
POST /api/admin/nodes/{id}/frps-reload
POST /api/admin/nodes/{id}/nginx-test
POST /api/admin/nodes/{id}/nginx-reload
POST /api/nodes/bind
```

### 5. 防火墙建议

节点服务器 `8090/tcp` 是管理代理端口，生产环境只允许控制面服务器 IP 访问。

```text
允许：控制面服务器 IP -> 节点服务器 8090/tcp
拒绝：其他公网 IP -> 节点服务器 8090/tcp
```
