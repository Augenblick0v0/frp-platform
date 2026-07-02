# 实现进度

## 已完成

- Monorepo 目录结构。
- 规划文档归档到 `docs/plans`。
- Go API Server 基础骨架。
- PostgreSQL 迁移和 SQLStore 持久化骨架。
- `DATABASE_URL` 启用 PostgreSQL，未设置时使用内存存储。
- Cloudflare 风格 Admin/User/Client WebUI 静态页面。
- 客户端 frpc 配置渲染器。
- 客户端本地 API：状态、日志、同步配置、启动/停止 frpc。
- 客户端 frpc 进程管理骨架。
- 用户面板真实 API 对接：注册、登录、兑换、创建隧道、列表刷新。
- 管理后台真实 API 对接：仪表盘、用户、套餐、隧道、设置、兑换码生成。
- 后端管理接口：用户列表、套餐创建、兑换码列表与批量生成。
- SMTP Mailer 与验证码邮件模板。
- 发送验证码时调用 Mailer，未配置 SMTP 时使用日志 dry-run。
- 管理后台测试邮件接口和按钮。
- Docker Compose 自建 mail-server 服务和 DNS/账号配置说明。
- Docker Compose、Nginx、frps 部署模板。
- Go 客户端骨架。

## 后续继续

- Redis 验证码和任务队列。
- 服务端 frps 管理与真实证书/Nginx 自动化。
- Let's Encrypt 自动证书申请。
- Windows 安装包构建。
