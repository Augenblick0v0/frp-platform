# FRP 商业化内网穿透平台规划文档

本目录包含 FRP 商业化内网穿透平台的一次性完整规划文档。

## 文件清单

| 文件 | 内容 |
|---|---|
| `01-PRD.md` | 完整产品需求文档 |
| `02-ARCHITECTURE.md` | 总体架构设计 |
| `03-DATABASE-DESIGN.md` | PostgreSQL 数据库设计 |
| `04-API-DESIGN.md` | API 接口设计 |
| `05-DOCKER-DEPLOYMENT.md` | Docker/Nginx/frps 部署方案 |
| `06-CLIENT-DESIGN.md` | Linux/Windows 客户端设计 |
| `07-DEVELOPMENT-TASKS.md` | 开发任务清单和里程碑 |
| `08-ACCEPTANCE-CHECKLIST.md` | 验收清单 |

## 已确认关键决策

- 第一版单节点。
- 服务端 Linux Docker 部署。
- 服务端带 WebUI 管理后台。
- 客户端支持 Linux WebUI 和 Windows 安装包。
- 客户端可以创建隧道。
- TCP/UDP 端口由系统从后台端口范围自动分配。
- HTTP/HTTPS 使用用户绑定域名访问，不带端口。
- HTTPS 证书由平台通过 Let's Encrypt 自动申请。
- Nginx 统一处理 80/443 反代。
- 邮件服务器由系统自建，用平台域名发验证码。
- 套餐中配置协议权限、域名权限、流量、限速和隧道数量。
- 前端统一采用 Cloudflare 网页风格。

## 推荐阅读顺序

1. `01-PRD.md`
2. `02-ARCHITECTURE.md`
3. `03-DATABASE-DESIGN.md`
4. `04-API-DESIGN.md`
5. `05-DOCKER-DEPLOYMENT.md`
6. `06-CLIENT-DESIGN.md`
7. `07-DEVELOPMENT-TASKS.md`
8. `08-ACCEPTANCE-CHECKLIST.md`
