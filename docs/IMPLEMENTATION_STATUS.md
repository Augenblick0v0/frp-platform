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
- CNAME 检测 API。
- Nginx HTTPS 配置自动生成。
- Certbot Let's Encrypt 证书申请命令封装，支持 dry-run。
- Nginx test/reload 命令封装。
- 管理后台域名与证书操作入口。
- 套餐限制校验：总隧道数、各协议隧道数、域名数、流量超限禁止创建。
- 流量统计 API：客户端上报、用户查询、后台今日总流量。
- PostgreSQL traffic_logs 表和 subscription traffic_used_bytes 更新。
- 用户面板/客户端 WebUI 流量展示与上报入口。
- Windows 客户端打包脚本、PowerShell 构建脚本和 NSIS 安装器模板。
- Linux 客户端 tar.gz 打包脚本、systemd service 和 install.sh。
- 客户端构建支持携带官方 frpc 二进制，未提供时生成提示文件。
- 管理员登录与后台 API 鉴权。
- 默认管理员由 ADMIN_EMAIL/ADMIN_PASSWORD 配置。
- PostgreSQL admin_users/admin_sessions 迁移与 seed。
- 管理后台登录表单和 Bearer token 调用。
- Docker Compose、Nginx、frps 部署模板。
- Go 客户端骨架。

## 后续继续

- Redis 验证码和任务队列。
- 服务端 frps 管理、证书状态持久化和自动续期调度。
- Let's Encrypt 自动证书申请。
