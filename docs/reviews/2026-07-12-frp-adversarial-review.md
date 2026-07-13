# FRP 平台对抗式审查报告

审查基线：`2bfdeb52c50f6490ed80a9ce77780d7fe0a52fbc`
审查范围：Master API、Store/SQLStore、FRPS/Node Agent、FRPC、本地 API、Admin/User/Client WebUI、Docker Compose/Nginx/发布资产。
审查约束：本轮未修改业务代码、未写入部署配置、未操作线上服务。

## 证据摘要

| 检查项 | 结果 | 证据 |
|---|---|---|
| Go 单元测试 | PASS | `ALLOW_INSECURE_DEFAULTS=true go test ./apps/api-server/... ./client/frp-client/...` |
| Go 竞态测试 | PASS（现有测试集） | `go test -race ./apps/api-server/... ./client/frp-client/...` |
| Go vet | PASS | `go vet ./apps/api-server/... ./client/frp-client/...` |
| 三端生产构建 | PASS，但每端 JS 约 1.18–1.22 MB | `output/adversarial-review/00-baseline/frontend-builds.txt` |
| Compose 渲染 | PASS；node 示例有空变量警告 | `output/adversarial-review/00-baseline/compose-*` |
| 前端行为测试 | WEAK EVIDENCE | 未发现 React 单元/组件测试文件 |
| Playwright 视觉检查 | MISSING | 当前 root 环境 Chrome 因 sandbox 限制无法启动 |
| 供应链扫描 | MISSING | `govulncheck`、`gosec` 未安装，不能视为无漏洞 |

## Findings

### [P0] 工作树中的 fnOS 环境文件包含真实形态的生产凭据

位置：`deploy/.env.fnos`、`deploy/.env`；权限分别为 0600 和 0644；`.gitignore` 忽略这些文件。
证据：文件包含 `DATABASE_URL`、`POSTGRES_PASSWORD`、`REDIS_PASSWORD`、`FRP_TOKEN`、`EPAY_KEY` 等变量；报告不记录值。

影响：凭据会被备份、打包、日志采集、误上传或其他本机用户读取；0644 的标准环境文件还允许同机非特权用户读取弱默认值。支付、数据库、FRP 控制面凭据一旦复用会造成跨系统接管。

建议：立即轮换所有实际值；将运行时 secrets 移至权限 0600 的外部 secret store；发布脚本拒绝把 `.env*`、日志和运行目录纳入归档；启动前检查权限、占位值和密钥来源。

### [P1] 流量计量完全信任客户端主动上报，可绕过套餐流量限制

位置：`apps/api-server/internal/platform/server.go:clientTraffic`；`apps/api-server/internal/platform/store.go:ReportTraffic`；`apps/api-server/internal/platform/sql_store.go:1025-1055`；`client/frp-client/internal/clientcore/server.go:152-189`。

复现条件：用户拥有有效 token，停止或修改本地客户端 `/api/traffic/report` 调用即可。服务端没有从 FRPS/node-agent 获取独立字节计数，也没有要求单调序列号或签名证明。

影响：用户可以不报告流量，导致超流量限制、运营统计和计费均失真；伪造大值还可让本人被错误封禁或制造数据库压力。

建议：把 FRPS 侧按用户/隧道的计数作为权威来源；客户端上报只作为诊断数据；增加单调 offset、重复包去重、最大单次增量和异常告警；套餐限制在服务端启动/转发链路强制执行。

### [P1] SQLStore 的隧道数量/协议额度检查位于事务外，并发创建可越过套餐上限

位置：`apps/api-server/internal/platform/sql_store.go:317-347`、`977-1022`、`344-415`。

复现条件：同一用户同时提交多次 `POST /api/tunnels`。多个请求先读取相同计数，通过 `checkSQLTunnelLimits`，随后分别插入隧道。

影响：`max_tunnels`、协议上限和域名上限不是原子的；商业套餐权益可被并发绕过。现有 `-race` 不会发现数据库逻辑竞态，因为它只覆盖 Go 内存数据竞争。

建议：在同一事务中锁定用户订阅行或使用 advisory lock；计数和插入在同一隔离级别完成；增加 PostgreSQL 并发集成测试，验证 50 个并发请求最终只成功允许数量。

### [P1] API、Node Agent 和 Local API 使用 `http.ListenAndServe`，没有读写/空闲超时；JSON body 也没有统一上限

位置：`apps/api-server/cmd/server/main.go:42`、`apps/api-server/cmd/node-agent/main.go:59`、`client/frp-client/main.go:29`、`apps/api-server/internal/platform/server.go:1207-1215`、`client/frp-client/internal/clientcore/server.go:123/162/197/224/245`。

复现条件：建立慢速连接并持续发送不完整 HTTP 头或超大 JSON body。

影响：连接和 goroutine 长时间占用，可造成 slowloris、内存增长、代理连接池耗尽；测速和配置同步接口还会把请求体直接交给 decoder。

建议：改用 `http.Server`，设置 `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` 和 `MaxHeaderBytes`；所有 JSON 请求使用 `http.MaxBytesReader`、单对象解码后拒绝 trailing data，并按路由设置上限。

### [P1] 生产 Compose 仍使用可漂移的 `latest` 镜像

位置：`deploy/docker-compose.yml:24,134`、`deploy/docker-compose.node.yml:21`，以及 control/fNOS 变体中的 mail/frps 引用。

影响：相同 tag 的重部署可能拉到不同 FRPS/mail-server 版本，产生协议、配置或安全回归；发布无法复现，也没有镜像 digest 证据。

建议：发布清单固定版本和 digest，构建前做 SBOM/漏洞扫描，升级必须有兼容性测试和回滚 tag。

### [P2] 节点 bind token 长期有效且可重复绑定，绑定请求可改写 AgentURL 等控制面元数据

位置：`apps/api-server/internal/platform/server.go:992-1007`、`store.go:810-854`、`sql_store.go:739-760`、`cmd/node-agent/main.go:62-73`。

复现条件：获得一次 `bind_token` 后重复调用 `/api/nodes/bind`，可持续更新节点地址、入口域名和端口范围，并重新取得 `agent_token`。

影响：token 泄露后可把 Master 的节点运维请求导向攻击者控制的 AgentURL，或改变资源分配元数据；bind loop 每 60 秒重复发送长期 token，扩大暴露窗口。

建议：bind token 单次消费且设置过期时间；绑定后轮换并撤销旧 token；仅允许初始化时提交不可变节点身份，地址变更走管理员重新批准；AgentURL 使用 allowlist/HTTPS 校验。

### [P2] FRPC 配置和本地 token 文件写入不是原子更新，日志文件权限过宽

位置：`client/frp-client/internal/clientcore/manager.go:31-46,91-101,167-190`。

影响：进程中断或磁盘满可能留下半份 `frpc.toml`/token；`frpc.log` 以 0644 创建，可能泄露本地服务地址、错误和连接信息；同步失败时旧配置与 UI 状态可能不一致。

建议：临时文件写入、fsync、rename，并保留上一份可回滚配置；token/log 使用 0600；对日志做轮转和敏感字段脱敏。

### [P2] 本地 API 的公开读接口和无 Origin 请求扩大本机攻击面

位置：`client/frp-client/internal/clientcore/server.go:25-35,79-94,278-297`。

影响：`/api/status`、`/api/logs` 无 token 保护；`/api/local-token` 对无 `Origin` 请求直接放行。任何本机进程、恶意浏览器扩展或被 DNS rebinding 影响的页面都可读取运行状态/日志，后续再尝试 token 访问控制面。

建议：除健康检查外所有本地 API 默认要求 token；token 只通过明确的本地 UI session/一次性握手获取；严格校验 scheme、host、port 和 Host header；必要时改为 Unix socket/Windows named pipe。

### [P2] 隧道 start/stop 只改变数据库状态，不保证客户端运行态同步

位置：`apps/api-server/internal/platform/server.go:431-466`、`client/frp-client/internal/clientcore/config.go:50-55`。

影响：用户点击停止后，已运行的 frpc 进程仍可能继续暴露隧道，直到客户端再次同步并重启；UI 显示状态与公网真实状态不一致。

建议：定义控制语义：服务端状态、配置状态、运行态分开；start/stop 触发客户端拉取/热更新或明确显示“下次同步生效”；增加最终一致性状态和失败重试。

### [P2] 前端把用户 token 写入 localStorage，支付链接使用新窗口但没有显式安全属性

位置：`apps/shared/frontend/api/client.js:30-42`、`apps/client-webui/src/App.jsx:16-35`、`apps/user-web/src/App.jsx:218`。

影响：同源 XSS、浏览器扩展或本机共享 profile 可读取长期 token；支付 `target="_blank"` 未显式 `rel="noopener noreferrer"`，增加 opener 操纵风险。

建议：优先 HttpOnly/SameSite session；本地客户端使用受保护 OS secret store；支付链接使用固定 allowlist 并加 `rel`，禁止 API 返回任意跳转地址。

### [P3] 前端没有行为测试，构建产物过大且关键 UI 验收证据缺失

位置：`apps/*/package.json` 只有 `build/dev/preview`；三端 JS bundle 约 1.18–1.22 MB；Playwright 在当前 root 环境因 Chromium sandbox 未启动。

影响：认证过期、重复提交、错误状态、移动端溢出和敏感字段展示没有自动回归门禁；首屏性能和缓存成本偏高。

建议：补充 API contract + React/Playwright smoke；路由级动态 import 和 `manualChunks`；在非 root/启用 sandbox 的 CI runner 重新执行桌面/移动截图和可访问性检查。

## 计划完成度初判

| 范围 | 当前判定 | 原因 |
|---|---|---|
| PRD 基础功能和三端结构 | PARTIAL | 代码/构建存在，但部分运行语义和客户端闭环未证明 |
| 安全修复计划 | PARTIAL | 已有 token/CORS/session/默认值修复，但本报告新增流量信任、节点绑定、本地 API 和请求资源边界问题 |
| Store/SQLStore 等价 | WEAK EVIDENCE | 内存测试较全，SQL 并发/迁移/恢复没有集成证据 |
| FRPS/Node Agent 线上运维 | WEAK EVIDENCE | 静态配置存在，命令失败、超时、回滚和真实节点故障未演练 |
| 三端 UI | PARTIAL | 构建通过，缺少当前环境可复现的浏览器行为和移动端证据 |
| 发布与部署 | PARTIAL | Compose 可渲染，但 latest 镜像、环境文件处理和恢复流程仍有风险 |

## 尚未证明的项目

- PostgreSQL 并发额度、订单回调、端口分配和迁移升级。
- 真实 FRPS 字节计量与流量限制闭环。
- node-agent/FRPC 在断网、进程崩溃、磁盘满、证书失败时的恢复。
- 三端桌面/移动端 Playwright、可访问性和错误态回归。
- `govulncheck`、`gosec`、镜像 digest/SBOM 和依赖漏洞结果。
