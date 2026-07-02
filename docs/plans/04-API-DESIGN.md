# API 设计

## 1. 通用约定

Base URL：

```text
https://api.example.com
```

响应格式：

```json
{
  "success": true,
  "data": {},
  "message": "ok"
}
```

错误格式：

```json
{
  "success": false,
  "code": "SUBSCRIPTION_EXPIRED",
  "message": "套餐已过期"
}
```

认证：

```text
Authorization: Bearer <token>
```

## 2. 用户认证 API

### 2.1 发送邮箱验证码

```http
POST /api/auth/send-email-code
```

请求：

```json
{
  "email": "user@example.com",
  "purpose": "register"
}
```

响应：

```json
{
  "success": true,
  "data": {
    "expires_in": 600
  }
}
```

### 2.2 注册

```http
POST /api/auth/register
```

请求：

```json
{
  "email": "user@example.com",
  "code": "123456",
  "password": "password123"
}
```

### 2.3 登录

```http
POST /api/auth/login
```

请求：

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

响应：

```json
{
  "success": true,
  "data": {
    "access_token": "jwt-token",
    "expires_in": 86400
  }
}
```

### 2.4 当前用户

```http
GET /api/auth/me
```

## 3. 用户套餐 API

### 3.1 获取当前订阅

```http
GET /api/user/subscription
```

响应：

```json
{
  "success": true,
  "data": {
    "plan_name": "高级套餐",
    "expires_at": "2026-08-01T00:00:00Z",
    "traffic_limit_bytes": 107374182400,
    "traffic_used_bytes": 1024,
    "bandwidth_limit_kbps": 10240,
    "allow_tcp": true,
    "allow_udp": true,
    "allow_http": true,
    "allow_https": true,
    "allow_custom_domain": true
  }
}
```

### 3.2 兑换码兑换

```http
POST /api/user/redeem
```

请求：

```json
{
  "code": "ABCD-EFGH-1234"
}
```

### 3.3 购买信息

```http
GET /api/user/purchase-info
```

## 4. 隧道 API

### 4.1 创建隧道

```http
POST /api/tunnels
```

TCP 请求：

```json
{
  "name": "ssh",
  "type": "tcp",
  "local_host": "127.0.0.1",
  "local_port": 22
}
```

HTTP 请求：

```json
{
  "name": "web",
  "type": "http",
  "local_host": "127.0.0.1",
  "local_port": 8080,
  "domain": "app.user.com"
}
```

HTTPS 请求：

```json
{
  "name": "secure-web",
  "type": "https",
  "local_host": "127.0.0.1",
  "local_port": 8080,
  "domain": "app.user.com",
  "use_https": true
}
```

响应：

```json
{
  "success": true,
  "data": {
    "id": 1001,
    "public_url": "https://app.user.com",
    "status": "pending_domain_check"
  }
}
```

### 4.2 隧道列表

```http
GET /api/tunnels
```

### 4.3 启动隧道

```http
POST /api/tunnels/:id/start
```

### 4.4 停止隧道

```http
POST /api/tunnels/:id/stop
```

### 4.5 删除隧道

```http
DELETE /api/tunnels/:id
```

## 5. 客户端 API

### 5.1 客户端登录

```http
POST /api/client/login
```

### 5.2 心跳

```http
POST /api/client/heartbeat
```

请求：

```json
{
  "device_id": "device-uuid",
  "device_name": "my-windows-pc",
  "os": "windows",
  "version": "1.0.0"
}
```

### 5.3 拉取客户端配置

```http
GET /api/client/tunnels
```

响应：

```json
{
  "success": true,
  "data": {
    "server_addr": "frp.example.com",
    "server_port": 7000,
    "token": "runtime-token",
    "tunnels": [
      {
        "id": 1,
        "name": "web",
        "type": "http",
        "local_host": "127.0.0.1",
        "local_port": 8080,
        "custom_domains": ["app.user.com"]
      }
    ]
  }
}
```

### 5.4 上报日志

```http
POST /api/client/logs
```

### 5.5 上报流量

```http
POST /api/client/traffic
```

## 6. 管理后台 API

### 6.1 仪表盘

```http
GET /api/admin/dashboard
```

### 6.2 套餐管理

```http
GET    /api/admin/plans
POST   /api/admin/plans
PUT    /api/admin/plans/:id
DELETE /api/admin/plans/:id
```

### 6.3 兑换码

```http
GET  /api/admin/redeem-codes
POST /api/admin/redeem-codes
POST /api/admin/redeem-codes/batch
PUT  /api/admin/redeem-codes/:id/disable
```

### 6.4 用户

```http
GET /api/admin/users
GET /api/admin/users/:id
PUT /api/admin/users/:id/status
PUT /api/admin/users/:id/subscription
```

### 6.5 端口池

```http
GET /api/admin/port-pools
PUT /api/admin/port-pools
GET /api/admin/port-allocations
```

### 6.6 域名和证书

```http
GET  /api/admin/domains
POST /api/admin/domains/:id/check-cname
GET  /api/admin/certificates
POST /api/admin/certificates/:id/renew
```

### 6.7 系统设置

```http
GET /api/admin/settings
PUT /api/admin/settings
POST /api/admin/settings/test-mail
```
