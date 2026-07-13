# FRP Platform Adversarial Review Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `code-review` for findings discipline, `diagnosing-bugs` for every reproduced failure, `playwright` for browser verification, and `design-system` for the three frontend audits. Execute inline; do not dispatch subagents unless the user later requests delegation. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 对 FRP 商业平台的 Master API、FRPS 节点面、FRPC 客户端、三套前端和部署资产做完整对抗式审查，验证边界条件、异常输入、并发、性能、安全、线上故障风险与历史计划完成度，并产出按优先级排序的可复现问题和建设性改进建议。

**Architecture:** 审查按信任边界和运行链路展开，而不是只按目录扫代码：Admin/User/Client WebUI -> Master API -> Store/SQLStore -> Node Agent/FRPS/Nginx -> FRPC/local API -> 用户本地服务。每个结论必须同时说明触发条件、实际影响、证据、建议修复位置和回归测试；现有 `PASS`、`DONE`、发布标签和文件存在本身不作为完成证明。

**Tech Stack:** Go 1.24 workspace、`net/http`、PostgreSQL、Redis、frps/frpc、Docker Compose、Nginx、React 19、Vite 7、Ant Design 5、Playwright。

## Global Constraints

- 本轮只执行审查和验证，不修改业务代码、生产配置、数据库数据或线上服务状态。
- 所有动态攻击用例先在本地临时环境执行；远端/fnOS 只允许只读探测，任何写操作、重启、reload、证书申请和支付回调前再次获得用户确认。
- `Store` 与 `SQLStore` 必须分别验证；只覆盖内存实现的测试不能证明 PostgreSQL 生产路径正确。
- Admin Web、User Web、Client WebUI 三端分别审查，不能用其中一端的构建成功推断另外两端正确。
- 每条计划/PRD 要求只能标记为 `PROVEN`、`CONTRADICTED`、`PARTIAL`、`WEAK EVIDENCE` 或 `MISSING`。
- 每条问题按 `P0/P1/P2/P3` 分级，并记录 `位置、前置条件、复现步骤、实际结果、期望结果、影响、修复建议、回归测试`。
- 敏感值在证据中只保留变量名、长度、前后各两字符；不记录完整 token、密码、支付密钥、SMTP 密钥或证书私钥。
- 不把文档勾选、文件存在、接口返回 200、单次截图或一次健康检查当作端到端完成证据。
- 基线提交固定为执行开始时的 `git rev-parse HEAD`；若审查期间工作树发生变化，停止结论合并并重新记录差异。

---

## Current Baseline Signals

- 当前分支基线：`master`，检查时 HEAD 为 `2bfdeb5`，与 `origin/master` 对齐且工作树干净；执行审查时必须重新确认。
- 主要复杂文件：`server.go` 1239 行、`sql_store.go` 1135 行、`store.go` 1096 行、FRPC `speed.go` 771 行，优先检查职责耦合和错误路径。
- 当前有 52 个 Go `Test*` 函数，但没有 React 单元/组件测试文件；前端目前主要依赖构建和截图证据。
- `docs/plans/07-DEVELOPMENT-TASKS.md` 仍为全未勾选，`docs/IMPLEMENTATION_STATUS.md` 同时存在“后续继续”和乱码段落，而最终验收文档宣称大量 `PASS/DONE`；必须逐项重建证据链。
- 已有两轮安全修复计划，重点复核修复是否同时覆盖内存/SQL、真实部署和绕过路径，而不是重复相信旧结论。

## Review Artifacts

执行阶段创建以下只读审查产物，不改业务代码：

- `output/adversarial-review/00-baseline/`：提交、依赖、构建、测试、配置渲染原始输出。
- `output/adversarial-review/01-static/`：`go vet`、`govulncheck`、`gosec`、前端依赖审计结果。
- `output/adversarial-review/02-api/`：Master API 异常输入、认证授权、支付回调和 HTTP 语义证据。
- `output/adversarial-review/03-persistence/`：Store/SQLStore 一致性、事务和并发复现证据。
- `output/adversarial-review/04-node-client/`：node-agent、FRPS、FRPC、本地 API 和进程生命周期证据。
- `output/adversarial-review/05-performance/`：竞态、负载、资源泄漏、超时和故障注入结果。
- `output/adversarial-review/06-ui/`：三端桌面/移动截图、控制台错误、可访问性和关键交互记录。
- `docs/reviews/2026-07-12-frp-plan-traceability.md`：PRD/架构/API/任务/验收逐项证据矩阵。
- `docs/reviews/2026-07-12-frp-adversarial-review.md`：最终发现、风险排序和优化建议。

---

### Task 1: 固定审查基线和完整资产清单

**Files:**
- Inspect: `go.work`
- Inspect: `apps/api-server/**`
- Inspect: `client/frp-client/**`
- Inspect: `apps/admin-web/**`
- Inspect: `apps/user-web/**`
- Inspect: `apps/client-webui/**`
- Inspect: `apps/shared/frontend/**`
- Inspect: `deploy/**`
- Inspect: `scripts/**`
- Evidence: `output/adversarial-review/00-baseline/`

**Interfaces:**
- Produces: 一个固定 commit、文件清单、依赖版本、暴露端口、服务拓扑和测试入口。
- Gate: 后续全部证据必须能回溯到同一 commit。

- [ ] **Step 1: 记录 Git 与工具链基线**

```bash
mkdir -p output/adversarial-review/00-baseline
git status --short --branch | tee output/adversarial-review/00-baseline/git-status.txt
git rev-parse HEAD | tee output/adversarial-review/00-baseline/git-head.txt
git log --oneline -20 | tee output/adversarial-review/00-baseline/git-log.txt
go version | tee output/adversarial-review/00-baseline/go-version.txt
node --version | tee output/adversarial-review/00-baseline/node-version.txt
npm --version | tee output/adversarial-review/00-baseline/npm-version.txt
docker version | tee output/adversarial-review/00-baseline/docker-version.txt
docker compose version | tee output/adversarial-review/00-baseline/compose-version.txt
```

Expected: 工作树无未解释改动，所有后续输出记录同一 HEAD。

- [ ] **Step 2: 建立源码、路由、服务和端口清单**

```bash
git ls-files | sort > output/adversarial-review/00-baseline/tracked-files.txt
grep -RhnE 'HandleFunc|\.Handle\(' apps/api-server client/frp-client --include='*.go' > output/adversarial-review/00-baseline/http-routes.txt
grep -RhnE 'ports:|expose:|listen|server_port|bindPort|vhostHTTPPort|vhostHTTPSPort' deploy apps client --include='*.yml' --include='*.yaml' --include='*.toml' --include='*.go' > output/adversarial-review/00-baseline/network-surfaces.txt
```

Expected: Master API、node-agent、本地 FRPC API、Nginx、FRPS 与三端入口均有唯一归属。

- [ ] **Step 3: 记录现有测试覆盖入口，不先评价通过与否**

```bash
grep -Rhn '^func Test' apps/api-server client/frp-client --include='*_test.go' > output/adversarial-review/00-baseline/go-tests.txt
find apps -type f \( -iname '*test*' -o -iname '*spec*' \) -not -path '*/node_modules/*' > output/adversarial-review/00-baseline/frontend-tests.txt
```

Expected: 明确哪些组件只有构建检查、没有行为测试。

---

### Task 2: 重建 PRD、计划和验收的可追溯矩阵

**Files:**
- Inspect: `docs/plans/01-PRD.md`
- Inspect: `docs/plans/02-ARCHITECTURE.md`
- Inspect: `docs/plans/03-DATABASE-DESIGN.md`
- Inspect: `docs/plans/04-API-DESIGN.md`
- Inspect: `docs/plans/05-DOCKER-DEPLOYMENT.md`
- Inspect: `docs/plans/06-CLIENT-DESIGN.md`
- Inspect: `docs/plans/07-DEVELOPMENT-TASKS.md`
- Inspect: `docs/plans/08-ACCEPTANCE-CHECKLIST.md`
- Inspect: `docs/superpowers/plans/*.md`
- Inspect: `docs/FINAL_ACCEPTANCE_AUDIT.md`
- Inspect: `docs/FINAL_MEFRP_REDESIGN_ACCEPTANCE.md`
- Inspect: `docs/IMPLEMENTATION_STATUS.md`
- Create during execution: `docs/reviews/2026-07-12-frp-plan-traceability.md`

**Interfaces:**
- Produces: 每个显式需求到代码、测试、构建、运行证据的双向映射。
- Gate: 没有直接证据的项目不得沿用旧 `PASS/DONE`。

- [ ] **Step 1: 提取全部需求和旧状态**

矩阵列固定为：`Requirement ID | Source | Requirement | Claimed Status | Code Evidence | Test Evidence | Runtime Evidence | Current Verdict | Gap`。

- [ ] **Step 2: 对重复或冲突计划建立优先级**

判定顺序固定为：当前用户目标 > 最新专项计划 > PRD/架构/API 文档 > 旧开发任务清单 > 最终验收报告。冲突必须保留在矩阵中，不静默选择有利版本。

- [ ] **Step 3: 逐项核对关键业务闭环**

至少覆盖：注册/验证码/登录、套餐/兑换码/支付、TCP/UDP/HTTP/HTTPS 创建、端口分配与释放、域名唯一性、证书申请续期、流量核算、限速、节点运维、客户端同步与进程控制、Windows/Linux 打包、邮件、自建部署、备份恢复、升级回滚。

- [ ] **Step 4: 标记文档漂移**

单独列出乱码、过期路径 `D:/frpbusiness`、版本号不一致、未勾选任务与 `DONE` 冲突、仅文件存在即判定 `PASS` 的条目。

---

### Task 3: 静态质量、依赖和供应链审查

**Files:**
- Inspect: all Go modules and frontend lockfiles
- Inspect: `Dockerfile` and Compose image references
- Evidence: `output/adversarial-review/01-static/`

**Interfaces:**
- Produces: 编译器/分析器问题、已知漏洞、未固定依赖、镜像和构建链风险列表。

- [ ] **Step 1: 执行 Go 静态检查**

```bash
mkdir -p output/adversarial-review/01-static
go vet ./apps/api-server/... ./client/frp-client/... 2>&1 | tee output/adversarial-review/01-static/go-vet.txt
govulncheck ./apps/api-server/... ./client/frp-client/... 2>&1 | tee output/adversarial-review/01-static/govulncheck.txt
gosec -fmt=json -out=output/adversarial-review/01-static/gosec.json ./apps/api-server/... ./client/frp-client/...
```

Expected: 工具缺失也记录为审查环境缺口，不把“命令未运行”写成无问题。

- [ ] **Step 2: 审查前端依赖与构建配置**

```bash
for app in apps/admin-web apps/user-web apps/client-webui; do
  (cd "$app" && npm ci && npm audit --json) > "output/adversarial-review/01-static/$(basename "$app")-npm-audit.json" 2>&1
done
```

重点检查：运行依赖与开发依赖混放、缺少测试/lint、锁文件一致性、Vite 代理和环境变量注入、生产 source map、依赖漏洞实际可达性。

- [ ] **Step 3: 审查容器和发布供应链**

检查基础镜像是否固定 digest、是否以 root 运行、下载是否校验 SHA256、frpc/frps 二进制来源、发布包是否可复现、`.env`/私钥是否可能进入镜像或 Git、`latest` 标签和自动升级风险。

---

### Task 4: Master API 对抗式输入、认证与授权审查

**Files:**
- Inspect: `apps/api-server/internal/platform/server.go`
- Inspect: `apps/api-server/internal/platform/security.go`
- Inspect: `apps/api-server/internal/platform/password.go`
- Inspect: `apps/api-server/internal/platform/payment.go`
- Inspect: `apps/api-server/internal/platform/mailer.go`
- Inspect: `apps/api-server/internal/platform/automation.go`
- Inspect: `apps/api-server/internal/platform/node_agent_client.go`
- Inspect: `apps/api-server/internal/platform/speed_probe.go`
- Inspect: `apps/api-server/internal/platform/server_test.go`
- Evidence: `output/adversarial-review/02-api/`

**Interfaces:**
- Produces: 逐路由的 HTTP 方法、认证、资源归属、输入上限、超时、幂等和错误语义矩阵。

- [ ] **Step 1: 为所有路由建立访问控制表**

表列固定为：`Route | Methods | Anonymous/User/Admin/Node | Resource Ownership Check | Body Limit | Timeout | Rate Limit | Idempotency | Sensitive Output`。

- [ ] **Step 2: 设计通用恶意输入集**

每个 JSON 接口至少验证：空 body、畸形 JSON、重复字段、未知字段、超大 body、深层 JSON、负数、零、整数溢出、NaN/科学计数、超长字符串、控制字符、Unicode 同形字符、空白归一化、重复提交、错误 Content-Type、错误 HTTP method、取消连接和慢速请求。

- [ ] **Step 3: 审查认证和会话生命周期**

验证用户/管理员 session 生成、过期、禁用用户、退出/撤销、并发登录、token 重放、Bearer 解析、CORS 预检、Origin/Host 信任、暴力破解和验证码频率限制。

- [ ] **Step 4: 审查对象级授权**

交叉用户验证隧道 start/stop/delete、流量上报、证书申请、测速临时隧道、节点和管理员资源，确认 ID 枚举不会访问别人的对象；分别测试普通用户、过期套餐用户、禁用用户和管理员 token 混用。

- [ ] **Step 5: 审查支付回调**

验证签名规范化、参数重复、金额/币种/套餐绑定、订单状态机、回调重放、并发回调、先 return 后 notify、伪造客户端 IP、超时重试和日志中的敏感信息。

- [ ] **Step 6: 审查外部调用攻击面**

重点覆盖 CNAME/DNS、SMTP、Certbot、Nginx、node-agent、速度探测目标，检查 SSRF、DNS rebinding、私网/环回/链路本地地址、重定向、无限响应、TLS 校验、context timeout 和连接池耗尽。

---

### Task 5: Store/SQLStore 一致性、事务与并发审查

**Files:**
- Inspect: `apps/api-server/internal/platform/backend.go`
- Inspect: `apps/api-server/internal/platform/models.go`
- Inspect: `apps/api-server/internal/platform/store.go`
- Inspect: `apps/api-server/internal/platform/sql_store.go`
- Inspect: `apps/api-server/internal/platform/sql_migrations.go`
- Evidence: `output/adversarial-review/03-persistence/`

**Interfaces:**
- Produces: Backend 接口逐方法的内存/SQL 语义差异表和并发不变量测试清单。

- [ ] **Step 1: 建立双实现行为矩阵**

逐个 `Backend` 方法比较：输入归一化、返回错误、排序/分页、时间时区、空值、唯一约束、权限判断、删除语义、日志、副作用和事务边界。

- [ ] **Step 2: 识别必须原子的业务不变量**

至少包括：兑换码只能消费一次、订单只能激活一次、端口在同一节点/协议唯一、删除隧道释放端口、套餐/隧道数量限制不会被并发绕过、域名唯一、流量累加不丢失、临时测速隧道可回收、证书状态和配置文件一致。

- [ ] **Step 3: 运行竞态测试**

```bash
mkdir -p output/adversarial-review/03-persistence
ALLOW_INSECURE_DEFAULTS=true go test -race -count=10 ./apps/api-server/... 2>&1 | tee output/adversarial-review/03-persistence/api-race.txt
go test -race -count=10 ./client/frp-client/... 2>&1 | tee output/adversarial-review/03-persistence/client-race.txt
```

- [ ] **Step 4: 在临时 PostgreSQL 上做并发事务用例**

并行 50 次执行同一兑换码兑换、同一端口池分配、同一订单回调、同一域名绑定和同一隧道删除；要求成功数和最终数据库状态符合唯一业务结果，无重复资源、负计数或孤儿记录。

- [ ] **Step 5: 审查迁移与升级路径**

检查迁移幂等性、旧版本升级、约束补加失败、默认值回填、长事务锁表、回滚能力、备份恢复和时区/精度变化。

---

### Task 6: FRPS 节点面、Node Agent 与自动化命令审查

**Files:**
- Inspect: `apps/api-server/cmd/node-agent/main.go`
- Inspect: `apps/api-server/internal/platform/node_agent_client.go`
- Inspect: `apps/api-server/internal/platform/frps_manager.go`
- Inspect: `apps/api-server/internal/platform/automation.go`
- Inspect: `apps/api-server/internal/platform/cert_renewer.go`
- Inspect: `deploy/frps/frps.toml`
- Inspect: `deploy/nginx-node/**`
- Inspect: `deploy/nginx/**`
- Evidence: `output/adversarial-review/04-node-client/`

**Interfaces:**
- Produces: 节点控制面认证、命令执行、配置文件和服务状态转换风险清单。

- [ ] **Step 1: 审查 node-agent 暴露与认证**

验证空 token、弱 token、Header 变体、方法混淆、跨节点 token、绑定 token 重放、health 信息泄露、监听地址和 Docker 网络暴露。

- [ ] **Step 2: 审查命令与路径边界**

对域名、节点地址、日志行数、配置内容和证书路径做 shell metacharacter、路径穿越、符号链接、换行注入和超长输入检查；确认使用参数化 `exec.CommandContext`、命令超时和输出上限。

- [ ] **Step 3: 审查配置写入和 reload 原子性**

验证临时文件 -> fsync -> rename -> `nginx -t` -> reload 的顺序、失败回滚、并发证书申请、重复域名、磁盘满、只读文件系统和进程被杀时的状态。

- [ ] **Step 4: 审查证书生命周期**

覆盖 CNAME 未生效、ACME rate limit、过期续期、证书/私钥权限、证书归属、删除隧道后的证书、多个节点证书分发和系统时钟漂移。

---

### Task 7: FRPC 本地客户端、Local API 与进程生命周期审查

**Files:**
- Inspect: `client/frp-client/main.go`
- Inspect: `client/frp-client/internal/clientcore/config.go`
- Inspect: `client/frp-client/internal/clientcore/manager.go`
- Inspect: `client/frp-client/internal/clientcore/server.go`
- Inspect: `client/frp-client/internal/clientcore/speed.go`
- Inspect: `client/packaging/**`
- Inspect: `apps/client-webui/**`
- Evidence: `output/adversarial-review/04-node-client/`

**Interfaces:**
- Produces: 本地控制面、frpc 子进程、配置同步、测速服务和安装升级风险清单。

- [ ] **Step 1: 审查本地 API 信任模型**

验证监听地址、`/api/local-token`、Host/Origin/CORS、DNS rebinding、浏览器跨站请求、token 存储、日志泄露、GET 接口敏感信息和未授权读接口。

- [ ] **Step 2: 审查 frpc 进程状态机**

并发执行 start/stop/restart/sync，覆盖重复启动、启动中停止、僵尸进程、PID 复用、异常退出、父进程退出、Windows service/Linux systemd、配置损坏和 frpc 缺失。

- [ ] **Step 3: 审查配置渲染和写入**

对名称、域名、本地地址、本地端口、remote port、token 和带宽字段做 TOML 注入、换行、重复 proxy 名、IPv6、localhost/Unix socket、非法协议和超量隧道检查；验证原子写和旧配置回滚。

- [ ] **Step 4: 审查测速服务资源边界**

验证下载/上传/UDP 大小上限、并发任务数、超时、临时监听端口、连接断开、慢速客户端、内存峰值、goroutine 泄漏、清理幂等和 Master/Local 两端状态不一致。

- [ ] **Step 5: 审查打包、安装和升级**

验证 Windows 路径空格/UAC/防火墙、Linux systemd 用户和权限、配置目录 ACL、卸载残留、升级覆盖、二进制校验、失败回滚以及 WebUI 资源与客户端版本匹配。

---

### Task 8: 性能、容量和故障注入审查

**Files:**
- Inspect: all backend/client hot paths
- Evidence: `output/adversarial-review/05-performance/`

**Interfaces:**
- Produces: 容量拐点、资源上限、超时传播和恢复行为报告。

- [ ] **Step 1: 建立负载场景**

固定场景：100 并发登录、100 并发建隧道、1000 隧道列表、50 并发测速、100 并发流量上报、20 并发 node-agent 操作、支付回调突发和证书续期批次。

- [ ] **Step 2: 采集关键指标**

记录 p50/p95/p99、错误率、数据库连接数、goroutine、FD、RSS、CPU、GC、锁等待、SQL 慢查询和外部命令持续时间；每个测试必须有固定持续时间和清理步骤。

- [ ] **Step 3: 注入依赖故障**

分别模拟 PostgreSQL 慢/断连、SMTP 慢、DNS 超时、node-agent 500/挂起、Certbot 卡死、Nginx test 失败、frps 崩溃、磁盘满、只读目录和系统时间偏移。

- [ ] **Step 4: 审查恢复和退避**

确认请求有 deadline、重试只用于幂等操作、退避有抖动、不会重试风暴、资源清理最终一致、健康检查能区分存活/就绪、错误日志可定位且不泄密。

---

### Task 9: Docker、Nginx、FRPS 和生产运维审查

**Files:**
- Inspect: `deploy/docker-compose.yml`
- Inspect: `deploy/docker-compose.control.yml`
- Inspect: `deploy/docker-compose.node.yml`
- Inspect: `deploy/docker-compose.fnos.yml`
- Inspect: `deploy/.env*.example`
- Inspect: `deploy/PRODUCTION.md`
- Inspect: `deploy/SPLIT_DEPLOYMENT.md`
- Inspect: `deploy/nginx*/**`
- Inspect: `scripts/release.sh`
- Inspect: `scripts/verify-release.sh`
- Evidence: `output/adversarial-review/00-baseline/`

**Interfaces:**
- Produces: 配置渲染差异、暴露面、启动顺序、持久化、回滚和观测性问题清单。

- [ ] **Step 1: 渲染全部 Compose 变体**

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config > output/adversarial-review/00-baseline/compose-standard.txt
docker compose -f deploy/docker-compose.control.yml --env-file deploy/.env.control.example config > output/adversarial-review/00-baseline/compose-control.txt
docker compose -f deploy/docker-compose.node.yml --env-file deploy/.env.node.example config > output/adversarial-review/00-baseline/compose-node.txt
docker compose -f deploy/docker-compose.fnos.yml --env-file deploy/.env.example config > output/adversarial-review/00-baseline/compose-fnos.txt
```

- [ ] **Step 2: 审查生产默认值和网络边界**

检查弱默认值、空变量、端口绑定到 `0.0.0.0`、容器特权、Docker socket、root 用户、只读根文件系统、capabilities、内部服务误暴露、TLS、Nginx header、请求体上限和速率限制。

- [ ] **Step 3: 审查数据耐久性和灾难恢复**

明确 PostgreSQL、Redis、证书、Nginx 动态配置、FRPS 配置、操作日志和客户端包的 volume、备份频率、恢复演练、RPO/RTO、迁移前备份和回滚顺序。

- [ ] **Step 4: 审查可观测性和线上处置**

检查结构化日志、request ID、审计日志完整性、日志轮转、指标、告警、健康/就绪检查、容量告警、证书到期告警和 node-agent/frps 失联告警。

---

### Task 10: 用户逻辑、前端安全和 UI/UX 对抗式审查

**Files:**
- Inspect: `apps/shared/frontend/**`
- Inspect: `apps/user-web/src/App.jsx`
- Inspect: `apps/user-web/src/styles.css`
- Inspect: `apps/admin-web/src/App.jsx`
- Inspect: `apps/admin-web/src/styles.css`
- Inspect: `apps/client-webui/src/App.jsx`
- Inspect: `apps/client-webui/src/styles.css`
- Evidence: `output/adversarial-review/06-ui/`

**Interfaces:**
- Produces: 三端关键任务成功率、异常状态、响应式、可访问性和前端安全问题清单。

- [ ] **Step 1: 建立三端任务清单**

User：注册、登录、套餐、兑换、支付、建隧道、启动/停止/删除、测速、证书、客户端下载。Admin：用户/套餐/订单/兑换码/隧道/节点/支付/证书/日志。Client：API 配置、token、同步、start/stop/restart、测速、日志、错误恢复。

- [ ] **Step 2: 验证异常和中间状态**

每个关键操作覆盖 loading、空数据、部分失败、401/403/404/409/422/429/500、网络断开、超时、重复点击、返回后状态恢复、token 过期、表单保留和成功后刷新。

- [ ] **Step 3: 验证前端安全**

检查 token 存储、DOM XSS、危险 URL、`target=_blank`、敏感字段复制、错误详情泄露、跨端 local token 误发、缓存用户数据、日志渲染和 API base 可控导致的凭据外送。

- [ ] **Step 4: 用 Playwright 做桌面与移动验收**

固定视口：`1440x900`、`1024x768`、`390x844`、`360x800`。检查菜单、表格横向滚动、Drawer/Modal、长域名/长邮箱、中文换行、按钮禁用、焦点顺序、键盘操作和文本遮挡。

- [ ] **Step 5: 审查产品逻辑和信息架构**

确认用户看见的是任务和结果，而不是内部架构术语；危险操作有明确对象与二次确认；套餐限制、端口、域名、证书、测速和失败原因能被普通用户理解；后台高风险操作与只读信息有清晰区分。

---

### Task 11: 端到端链路和线上故障场景验证

**Files:**
- Inspect: full system
- Evidence: `output/adversarial-review/02-api/`, `04-node-client/`, `05-performance/`, `06-ui/`

**Interfaces:**
- Produces: 从 UI 到公网访问的真实链路证据，以及失败时的责任层定位。

- [ ] **Step 1: 本地隔离环境跑核心闭环**

注册 -> 验证码 -> 登录 -> 兑换套餐 -> 创建 TCP/UDP/HTTP/HTTPS -> FRPC 同步 -> 启动 -> Visitor 访问 -> 流量上报 -> 停止/删除 -> 端口/域名/临时资源释放。

- [ ] **Step 2: 验证跨层失败定位**

分别让 Master、PostgreSQL、node-agent、frps、Nginx、frpc 和本地服务失败，确认 UI/API 能指出故障层，且不会错误显示成功或留下不可恢复状态。

- [ ] **Step 3: 只读核对当前 fnOS 部署**

在用户确认进入执行阶段后，读取容器镜像、环境变量键名、端口、健康状态、配置 hash、日志尾部和公网 GET 结果；不重启、不 reload、不发证书、不创建订单、不改隧道。

- [ ] **Step 4: 对照发布包验证版本一致性**

比较 Git tag、容器二进制、前端静态资源、Windows/Linux 客户端包和 SHA256 清单，识别“源码已修但线上/包内仍旧”的漂移。

---

### Task 12: 汇总发现、优化建议和最终完成度结论

**Files:**
- Create during execution: `docs/reviews/2026-07-12-frp-adversarial-review.md`
- Finalize: `docs/reviews/2026-07-12-frp-plan-traceability.md`

**Interfaces:**
- Produces: 可直接进入修复排期的最终审查报告。

- [ ] **Step 1: 先列问题，按严重度排序**

每条发现采用固定格式：

```text
[P0-P3] 标题
位置：file:line
前置条件：
复现步骤：
实际结果：
影响：
修复建议：
回归测试：
证据：output/adversarial-review/...
```

- [ ] **Step 2: 单列计划完成度**

按 PRD、开发任务、专项计划、验收清单四层给出数量汇总：`PROVEN / CONTRADICTED / PARTIAL / WEAK EVIDENCE / MISSING`，并列出阻止发布的未证明项。

- [ ] **Step 3: 给出建设性优化路线图**

分为：`立即封堵（24h）`、`发布前（1 周）`、`稳定性阶段（2-4 周）`、`架构演进`。每项说明收益、成本、风险、依赖和建议 owner，不把大规模重构当作默认答案。

- [ ] **Step 4: 给出测试补齐清单**

明确新增单元、SQL 集成、并发、fuzz、契约、Playwright E2E、负载、故障注入和部署验收测试；每个测试绑定至少一条发现或计划要求。

- [ ] **Step 5: 做最终自审**

确认：所有结论有证据；所有 P0/P1 有复现；Store/SQLStore 都覆盖；三套前端都覆盖；所有计划要求有判定；未执行的检查明确写为缺口；报告未包含密钥。

---

## Severity Standard

- `P0`：可直接导致平台接管、跨租户数据/隧道控制、远程命令执行、支付资金错误或大面积生产中断。
- `P1`：较低门槛导致敏感信息泄露、套餐/计费绕过、持久数据破坏、稳定复现的拒绝服务或无法自动恢复的故障。
- `P2`：需要特定条件的安全/并发问题、明显性能退化、错误恢复不足、重要业务逻辑或 UI 流程缺陷。
- `P3`：可维护性、文档漂移、可观测性、可访问性、低影响体验问题和长期架构债务。

## Exit Criteria

- 所有仓库内一方代码、配置、脚本、发布和三端 UI 均进入审查范围。
- 所有公开/本地/节点 HTTP 路由都有方法、认证、授权、输入和超时判定。
- `Store` 与 `SQLStore` 的关键业务不变量都有证据。
- `go test -race`、静态安全检查、依赖检查、Compose 渲染和前端构建结果已记录。
- 核心业务闭环和至少九类依赖故障已验证，未执行项明确标注。
- PRD 和历史计划的每项要求都有当前判定，不沿用未经验证的 `PASS/DONE`。
- 最终报告以发现为主，包含用户逻辑、前端 UI、性能、安全、并发、线上故障和修复优先级建议。

## Self-Review

- Spec coverage: 已覆盖用户要求的全部代码、边界情况、异常输入、并发、性能、安全、线上故障、计划完成度、用户逻辑和前端 UI。
- Scope discipline: 当前只创建审查计划；业务代码、部署状态和线上数据均不变。
- Evidence discipline: 每个工作流都有明确文件、命令、产物、判定标准和退出条件。
- Placeholder scan: 无待补内容；执行时使用当前 HEAD、当前依赖和实际环境值写入证据。
