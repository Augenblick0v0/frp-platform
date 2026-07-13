# FRP 平台修改与优化实施计划

> 本计划建立在 `2026-07-12-frp-adversarial-review.md` 的实际发现之上。执行前需先确认凭据轮换范围和是否允许本地/隔离 PostgreSQL、浏览器及节点测试；本计划本身不执行修改。

## 阶段 0：发布冻结与证据保护

**目标：** 防止审查期间的配置、凭据和版本继续漂移。

- [ ] 记录基线 commit、容器/发布包 hash 和当前环境变量键名；敏感值只记录 hash/长度。
- [ ] 轮换 `deploy/.env.fnos` 中所有数据库、Redis、FRP、支付和管理员凭据；把运行文件迁移到 0600 secret 目录。
- [ ] 修改 release 脚本，明确排除 `.env*`、日志、数据库数据、证书私钥和运行目录，并新增发布包泄密扫描。
- [ ] 为标准/control/node/fnOS Compose 固定 FRPS、mail-server、Postgres、Redis 镜像版本和 digest。
- [ ] 验收：旧凭据失效；`git archive`、发布 tar/zip、Docker build context 均不含 secrets；Compose 无 `latest`、空必需变量或弱默认值。

## 阶段 1：P0/P1 安全与资源边界

### 1.1 HTTP 资源上限与超时

**修改位置：** `apps/api-server/cmd/server/main.go`、`apps/api-server/internal/platform/server.go`、`apps/api-server/cmd/node-agent/main.go`、`client/frp-client/main.go`、两个 local server helper。

- [ ] 引入统一 `http.Server`，设置 header/read/write/idle timeout 与最大 header。
- [ ] 为 JSON 路由增加按路由的 `MaxBytesReader`；decoder 后使用第二次 decode 拒绝多余 JSON。
- [ ] 为测速、证书、node-agent、SMTP、FRPS 命令统一传播 context deadline。
- [ ] 测试：慢 header、超大 body、断开连接、慢外部命令、超时后 goroutine/FD 无持续增长。

### 1.2 可信流量计量

**修改位置：** `apps/api-server/internal/platform/store.go`、`sql_store.go`、`sql_migrations.go`、`server.go`、FRPS/node-agent 采集接口。

- [ ] 定义权威计量源和口径：按用户、隧道、方向、时间窗口记录；客户端上报降级为诊断。
- [ ] 增加计量快照、单调序列/去重键、最大增量和异常告警字段。
- [ ] 在服务端/FRPS 运行链路上阻止超套餐隧道继续转发，不只在 UI 或创建接口检查。
- [ ] 测试：禁用客户端上报、重复上报、负数/溢出、大批量上报、跨用户 tunnel ID、断线补报和计量恢复。

### 1.3 SQL 事务内套餐额度检查

**修改位置：** `apps/api-server/internal/platform/sql_store.go:317-415,977-1022`、迁移与数据库集成测试。

- [ ] 将订阅锁定、额度计数、端口分配、域名唯一检查和隧道插入放入同一事务。
- [ ] 使用 `SELECT ... FOR UPDATE` 或按 user/subscription 的 advisory lock；避免普通查询后再插入的 TOCTOU。
- [ ] 内存 Store 保持同一业务语义，不用单独更宽松的逻辑。
- [ ] 测试：50 并发建隧道、域名、兑换码、订单回调；断言最终成功数、端口数、套餐状态和审计记录。

## 阶段 2：P2 控制面和客户端一致性

### 2.1 节点绑定身份与轮换

**修改位置：** `server.go:nodeBind`、`store.go`、`sql_store.go`、`models.go`、`cmd/node-agent/main.go`。

- [ ] bind token 增加过期、单次消费和节点身份绑定；成功后立即轮换 agent token。
- [ ] 初始化绑定只允许必要元数据；AgentURL/端口范围变更必须经过管理员批准并校验 scheme/host/allowlist。
- [ ] 停止 bind loop 重复发送长期 token，改为短期心跳凭据或已签名节点身份。
- [ ] 测试：重复 bind、过期 bind、重放、恶意 AgentURL、跨节点 token 和断线重连。

### 2.2 本地 API 和 FRPC 文件安全

**修改位置：** `client/frp-client/internal/clientcore/server.go`、`manager.go`、`main.go`、`apps/client-webui/src/App.jsx`。

- [ ] 除 `/health` 外的状态/日志接口统一鉴权；token 获取改为一次性本地握手，并严格校验 Origin/Host/scheme/port。
- [ ] 优先使用 OS secret store；至少将 token/log/config 文件权限收紧到 0600。
- [ ] 配置和 token 使用临时文件、fsync、rename、备份和回滚；日志轮转并限制总大小。
- [ ] FRPC start/stop/restart 增加明确状态机、等待退出、PID 校验和失败恢复。
- [ ] 测试：并发 start/stop/sync、磁盘满、进程被杀、配置损坏、token 重建、日志超限和多用户本机访问。

### 2.3 隧道状态闭环

**修改位置：** `server.go:tunnelAction`、`clientTunnels`、`clientcore/config.go`、前端隧道操作组件。

- [ ] 明确 `desired_status`、`config_status`、`runtime_status` 三种状态。
- [ ] start/stop/delete 触发客户端同步或提供可见的“等待客户端同步”状态；客户端配置只包含允许运行的隧道。
- [ ] 测试：服务端操作后不重启客户端、同步失败、frpc 热更新失败、重试和最终恢复。

## 阶段 3：前端用户逻辑、UI 和性能

**修改位置：** `apps/shared/frontend/api/client.js`、`apps/user-web/src/App.jsx`、`apps/admin-web/src/App.jsx`、`apps/client-webui/src/App.jsx`、三端 styles/vite 配置。

- [ ] token 改为 HttpOnly/SameSite session；本地客户端使用 OS secret store 或受保护桥接，不把长期凭据放入 `localStorage`。
- [ ] 支付 URL 只允许配置白名单并渲染 `rel="noopener noreferrer"`；所有危险操作增加对象、影响和确认态。
- [ ] 为所有关键操作补 loading/empty/401/403/409/422/429/500/timeout/offline/重复点击状态，且成功后刷新真实状态。
- [ ] 将客户端、管理端、用户端路由做动态 import 和合理 manual chunks；目标首屏主 chunk < 500 KB gzip 前再评估是否拆包。
- [ ] 添加 React 组件测试和 Playwright smoke：注册/登录、兑换、建隧道、启停删、测速、节点操作、移动端菜单、长文本和键盘焦点。
- [ ] 重新设计用户端文案，使套餐限制、计量来源、同步延迟和失败层级可理解；后台危险运维按钮显示影响范围和回滚方式。

## 阶段 4：验证、部署和回滚

- [ ] 安装并运行 `govulncheck`、`gosec`、`npm audit`、镜像扫描和 SBOM 生成。
- [ ] 在隔离 PostgreSQL 上执行迁移、并发、备份恢复、版本升级和回滚演练。
- [ ] 在非 root、启用 Chromium sandbox 的 CI runner 执行桌面/移动 Playwright 与截图 diff。
- [ ] 执行 `go test ./...`、`go test -race ./...`、`go vet ./...`、三端 `npm ci && npm run build`、四套 Compose `config` 和发布包校验。
- [ ] 先部署 staging，验证 health/readiness、FRPS/FRPC 真实连接、TCP/UDP/HTTP/HTTPS、证书、流量计量和节点故障恢复，再生成生产变更窗口。
- [ ] 生产发布必须有数据库备份、配置备份、旧镜像/二进制、迁移回滚步骤和明确停止条件；本计划不授权执行这些变更。

## 优先级出口

| 优先级 | 必须完成 |
|---|---|
| P0 | 凭据轮换、secret 文件隔离、发布包泄密门禁 |
| P1 | 流量权威计量、SQL 额度事务化、HTTP/body 资源边界、镜像固定 |
| P2 | 节点绑定轮换、本地 API/FRPC 文件安全、隧道状态闭环、token 存储和 UI 错误态 |
| P3 | 前端拆包、可访问性、设计一致性、文档/计划状态收敛 |
