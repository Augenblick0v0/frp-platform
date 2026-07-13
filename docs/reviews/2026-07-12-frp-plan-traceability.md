# FRP 计划完成度追溯矩阵

基线：`2bfdeb52c50f6490ed80a9ce77780d7fe0a52fbc`。判定只接受代码、测试、构建、运行或发布证据；文档勾选本身不构成证明。

| 需求/计划域 | 当前判定 | 证据 | 差距 |
|---|---|---|---|
| Master API 路由和认证 | PARTIAL | `server.go` 注册用户、管理、支付、测速、节点路由；Go API 测试通过 | 无统一 body 上限/HTTP timeout；部分 handler 未在入口统一限制方法 |
| 用户注册、登录、验证码 | PARTIAL | `server_test.go` 覆盖验证码、token、session、禁用用户 | 无真实 SMTP、限流、重放和高并发证据 |
| 套餐、兑换码、支付 | PARTIAL | 兑换/订单/签名单测和 UI 存在 | 流量计量客户端可信；SQL 订单/兑换并发集成缺失 |
| TCP/UDP/HTTP/HTTPS 隧道 | PARTIAL | Store/SQLStore 创建逻辑、FRPC renderer 测试 | 客户端运行态与 start/stop 状态未闭环；真实公网链路未证明 |
| 端口分配与释放 | PARTIAL | SQL unique `(node_id, protocol, port)`、释放代码、单测 | 并发额度与多节点 PostgreSQL 集成证据缺失 |
| 域名/证书/Nginx | PARTIAL | Automation、证书测试、配置模板 | 原子写、失败回滚、ACME 限流和真实节点故障未演练 |
| 流量统计/套餐上限 | CONTRADICTED | `clientTraffic` 接受用户 token 下的主动 reports | 用户可停止上报，当前实现不能证明商业限额有效 |
| FRPS/Node Agent 运维 | PARTIAL | node-agent 路由、FRPS manager、Admin node tests | bind token 长期复用；命令、地址和恢复边界未充分约束 |
| FRPC 本地客户端 | PARTIAL | config/manager/local API 单测和构建通过 | local API 读接口、文件原子性、进程异常和安装升级缺少集成证据 |
| Admin/User/Client 前端 | PARTIAL | 三端 `npm run build` 通过，已有截图资产 | 无前端行为测试；本环境 Playwright Chromium sandbox 启动失败 |
| Docker/Compose 部署 | PARTIAL | 四套 Compose 可渲染 | `latest` 镜像、示例空变量、secret 隔离、备份恢复和 digest 未完成 |
| 发布包与版本一致性 | WEAK EVIDENCE | Git tag、release/verify 脚本存在 | 当前审查未重新验证所有包、镜像和线上 hash 一致 |
| 原始开发任务清单 `07-DEVELOPMENT-TASKS.md` | CONTRADICTED | 文件仍保留全未勾选任务 | 与最终验收报告 `DONE/PASS` 冲突，需要统一状态来源 |
| 安全修复专项计划 | PARTIAL | 旧计划对应修复提交和测试存在 | 本次新增 P0/P1/P2 风险未纳入旧验收门禁 |

## Evidence Gaps

- `govulncheck`、`gosec` 未安装，依赖/静态安全扫描尚未形成证据。
- PostgreSQL 实例并发、迁移、备份恢复和故障注入尚未执行。
- 真实 FRPS 字节计量、客户端断网、node-agent 失联、证书失败和公网访问闭环尚未执行。
- 浏览器视觉/交互审查受当前 root Chromium sandbox 限制，必须在合适 CI runner 重跑。
