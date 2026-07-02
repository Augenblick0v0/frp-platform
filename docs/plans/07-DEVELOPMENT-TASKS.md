# 开发任务清单

## Milestone 1：项目骨架

- [ ] 创建 monorepo。
- [ ] 创建 api-server 项目。
- [ ] 创建 admin-web 项目。
- [ ] 创建 user-web 项目。
- [ ] 创建 client 项目。
- [ ] 创建 deploy/docker-compose.yml。
- [ ] 接入 PostgreSQL。
- [ ] 接入 Redis。
- [ ] 实现 health check。

验收：

```text
Docker Compose 可以启动 api-server、postgres、redis、nginx、frps。
/api/health 返回 ok。
```

## Milestone 2：用户和认证

- [ ] users 表迁移。
- [ ] email_verification_codes 表迁移。
- [ ] 实现发送验证码接口。
- [ ] 实现注册接口。
- [ ] 实现登录接口。
- [ ] 实现 JWT 鉴权。
- [ ] 实现当前用户接口。

验收：

```text
用户可以通过邮箱验证码注册并登录。
```

## Milestone 3：邮件服务器

- [ ] docker-compose 增加 mail-server。
- [ ] 后台增加 SMTP 设置。
- [ ] 实现测试邮件发送。
- [ ] 实现验证码邮件模板。
- [ ] 实现邮件发送日志。
- [ ] 后台展示 SPF/DKIM/DMARC 配置提示。

验收：

```text
平台可以使用 noreply@yourdomain.com 发送验证码。
```

## Milestone 4：套餐和兑换码

- [ ] plans 表迁移。
- [ ] subscriptions 表迁移。
- [ ] redeem_codes 表迁移。
- [ ] 后台套餐 CRUD。
- [ ] 后台兑换码生成。
- [ ] 后台批量生成兑换码。
- [ ] 用户兑换接口。
- [ ] 用户订阅查询接口。
- [ ] 购买链接配置。

验收：

```text
管理员创建套餐和兑换码，用户可以兑换套餐。
```

## Milestone 5：TCP/UDP 隧道

- [ ] tunnels 表迁移。
- [ ] port_allocations 表迁移。
- [ ] 后台端口池配置。
- [ ] 实现空闲端口分配。
- [ ] 实现 TCP 隧道创建。
- [ ] 实现 UDP 隧道创建。
- [ ] 实现隧道删除释放端口。
- [ ] 客户端拉取 frpc 配置。
- [ ] 客户端启动 frpc。

验收：

```text
用户创建 TCP/UDP 隧道后，系统自动分配端口并可连接本地服务。
```

## Milestone 6：HTTP/HTTPS 隧道

- [ ] domains 表迁移。
- [ ] certificates 表迁移。
- [ ] 实现域名唯一绑定。
- [ ] 实现 CNAME 检测。
- [ ] 实现 HTTP 隧道创建。
- [ ] 实现 HTTPS 隧道创建。
- [ ] 实现 Let's Encrypt 证书申请。
- [ ] 实现 Nginx 配置生成。
- [ ] 实现 nginx -t 校验。
- [ ] 实现 Nginx reload。

验收：

```text
用户绑定域名 CNAME 后，可以通过 http://domain 和 https://domain 不带端口访问本地服务。
```

## Milestone 7：管理后台

- [ ] 仪表盘。
- [ ] 用户管理。
- [ ] 套餐管理。
- [ ] 兑换码管理。
- [ ] 购买链接设置。
- [ ] 端口池管理。
- [ ] 隧道管理。
- [ ] 域名管理。
- [ ] 证书管理。
- [ ] 邮件设置。
- [ ] 操作日志。

验收：

```text
管理员可以完成平台所有运营配置。
```

## Milestone 8：客户端

- [ ] 客户端登录。
- [ ] 客户端注册。
- [ ] 客户端兑换套餐。
- [ ] 客户端隧道列表。
- [ ] 客户端新增隧道。
- [ ] 客户端启动/停止隧道。
- [ ] 客户端日志展示。
- [ ] Linux WebUI 打包。
- [ ] Windows 安装包构建。

验收：

```text
Linux 和 Windows 用户都可以通过客户端完成注册、登录、兑换、创建隧道和查看日志。
```

## Milestone 9：流量统计

- [ ] traffic_logs 表迁移。
- [ ] 客户端上报流量。
- [ ] 服务端聚合流量。
- [ ] 用户端展示流量。
- [ ] 后台展示流量。
- [ ] 超流量禁止启动隧道。

验收：

```text
平台可以显示用户和隧道流量，并在超限后阻止继续使用。
```

## Milestone 10：验收和发布

- [ ] 完整部署测试。
- [ ] 注册登录测试。
- [ ] 邮件发送测试。
- [ ] 套餐兑换测试。
- [ ] TCP 穿透测试。
- [ ] UDP 穿透测试。
- [ ] HTTP 域名测试。
- [ ] HTTPS 证书测试。
- [ ] Windows 安装包测试。
- [ ] Linux 客户端测试。
