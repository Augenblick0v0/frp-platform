# API 托管隧道测速与用户控制台重设计 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用户在云端用户控制台选择节点/协议后，一键创建受套餐限速约束的临时隧道，由 API Server 发起真实测速，并把用户端控制台重设计为匹配 mefrp 信息架构但保持 Cloudflare 风格的独立容器。

**Architecture:** 保持三端分离：后台管理容器只做运营管理，用户控制台容器承载用户动线，本地 Win/Linux 客户端负责 frpc 与本地临时测速服务。用户控制台协调本地客户端与 API Server：本地客户端准备临时 benchmark 服务，API Server 创建临时隧道、触发探测、记录结果和流量，完成后清理。测速结果明确标注真实隧道速度受“套餐限速、节点能力、用户本地宽带”三者共同影响。

**Tech Stack:** Go API Server、Go 本地客户端、原生 HTML/CSS/JavaScript 用户控制台、PostgreSQL/内存 Store 双实现、Docker Compose 飞牛部署。

## Global Constraints

- 默认中文文案；代码、命令、API 字段保留英文。
- 后台容器、用户控制台容器、本地 Win/Linux 客户端保持独立，不把后台与用户端合并。
- 测速入口不允许用户自定义超过套餐的限速；测速隧道默认继承套餐限速。
- HTTP/HTTPS/TCP/UDP 四类测速均提供用户可调用入口。
- mefrp 只作为信息架构和动线参考，不复制代码、不复制视觉；视觉使用 Cloudflare 风格。
- 节点捐赠、广告投放、权益抽取、实名认证不进入本期实现；域名管理按证书/域名能力保留。

---

### Task 1: 领域语言与架构记录

**Files:**
- Modify: `D:/frpbusiness/CONTEXT.md`
- Create: `D:/frpbusiness/docs/adr/0001-api-managed-speed-test.md`

**Interfaces:**
- Produces: 统一术语 `API-Managed Speed Test`、`Local Client Benchmark Service`、`User Console`。

- [ ] **Step 1: 更新 CONTEXT.md**

新增术语：API 托管测速、本地客户端测速服务、用户控制台。

- [ ] **Step 2: 写 ADR**

记录为什么选择“用户控制台协调 + API Server 发起探测 + 本地客户端只准备服务”的方案。

- [ ] **Step 3: Commit**

```bash
git add CONTEXT.md docs/adr/0001-api-managed-speed-test.md docs/superpowers/plans/2026-07-09-api-managed-speedtest-user-console.md
git commit -m "docs: plan api managed speed tests"
```

### Task 2: 后端 API 托管测速

**Files:**
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/models.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/backend.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/server.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/sql_store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`

**Interfaces:**
- Consumes: `SpeedTestTunnelRequest{type, local_host, local_port}`。
- Produces: `POST /api/speed-tests/run`，请求 `SpeedTestRunRequest`，返回 `SpeedTestRunResult`。

- [ ] **Step 1: 写 failing tests**

覆盖：用户不能传测速限速；API Server 创建临时隧道、探测、上报流量、自动清理；结果包含瓶颈说明。

- [ ] **Step 2: 后端模型**

新增 `SpeedTestRunRequest`、`SpeedTestRunResult`、`SpeedTestProbeMetrics`、`SpeedTestBottleneckHint`。

- [ ] **Step 3: API Server 探测实现**

复用/迁移本地客户端已有 HTTP/TCP/UDP probe 思路，API Server 对临时隧道 public_url 发起探测。

- [ ] **Step 4: 套餐限速约束**

`CreateSpeedTestTunnel` 强制 `BandwidthKbps=0`，只继承套餐，不接受用户自定义测速限速。

- [ ] **Step 5: 测试**

```bash
go test ./apps/api-server/internal/platform -run SpeedTest -v
go test ./apps/api-server/...
```

### Task 3: 本地客户端协调能力

**Files:**
- Modify: `D:/frpbusiness/client/frp-client/internal/clientcore/server.go`
- Modify: `D:/frpbusiness/client/frp-client/internal/clientcore/speed.go`
- Modify: `D:/frpbusiness/client/frp-client/internal/clientcore/manager_test.go`
- Modify: `D:/frpbusiness/apps/client-webui/index.html`

**Interfaces:**
- Produces: `POST /api/speed-tests/prepare` 返回 `{type, local_host, local_port, expires_at}`。
- Produces: `POST /api/frpc/restart` 供用户控制台同步临时隧道后启动。

- [ ] **Step 1: CORS**

本地客户端允许用户控制台从局域网域名调用 `127.0.0.1:18080`。

- [ ] **Step 2: 准备本地 benchmark 服务**

把当前 `startBenchmarkService` 暴露成生命周期可控的 `PrepareSpeedBenchmark`。

- [ ] **Step 3: 重启接口**

新增本地 `/api/frpc/restart`，用户控制台同步配置后可启动临时隧道。

- [ ] **Step 4: 测试**

```bash
go test ./client/frp-client/...
```

### Task 4: 用户控制台重设计

**Files:**
- Modify: `D:/frpbusiness/apps/user-web/index.html`
- Modify: `D:/frpbusiness/apps/user-web/workbench.html`
- Modify: `D:/frpbusiness/apps/user-web/style.css`

**Interfaces:**
- Consumes: `/api/auth/me`、`/api/user/subscription`、`/api/user/traffic`、`/api/tunnels`、`/api/admin? no` 不使用后台接口。
- Produces: 统一用户控制台导航：控制面板、创建隧道、隧道管理、节点监控、客户端下载、帮助中心、用户中心、订单与支付、更多/域名证书、隧道测速。

- [ ] **Step 1: Cloudflare 风格信息架构**

保留单独用户端容器，登录后进入统一用户控制台，不合并后台。

- [ ] **Step 2: mefrp 动线映射**

按 mefrp 的分类/顺序改造导航和页面内容，但使用本项目术语：套餐、隧道、节点、证书、订单。

- [ ] **Step 3: 测速页**

选择协议、选择节点/默认节点、连接本地客户端、准备临时服务、同步 frpc、调用 API Server 测速、展示下载/上传/延迟/套餐限速占比/瓶颈说明。

- [ ] **Step 4: 移除测速限速输入**

前端不再允许用户设置测速限速。

- [ ] **Step 5: JS 语法检查**

```bash
python3 - <<'PY'
from pathlib import Path
for name in ['index','workbench']:
    html=Path(f'apps/user-web/{name}.html').read_text(encoding='utf-8')
    script=html.split('<script>',1)[1].split('</script>',1)[0]
    Path(f'/tmp/user-{name}.js').write_text(script,encoding='utf-8')
PY
node --check /tmp/user-index.js
node --check /tmp/user-workbench.js
```

### Task 5: 飞牛部署与真实全流程测试

**Files:**
- Modify as needed: `D:/frpbusiness/deploy/docker-compose.fnos.yml`
- Modify as needed: `D:/frpbusiness/deploy/.env.fnos` only on server, not committed if secrets involved。

**Interfaces:**
- Produces: 飞牛线上用户端、后台端、API Server 正常运行。

- [ ] **Step 1: 本地测试**

```bash
go test ./apps/api-server/...
go test ./client/frp-client/...
```

- [ ] **Step 2: 推送/同步飞牛**

按现有 bundle 流程或直接 git 流程同步到 `/root/frp-platform`。

- [ ] **Step 3: 部署**

```bash
cd /root/frp-platform
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/fnos/api-server ./apps/api-server/cmd/server
docker compose -f deploy/docker-compose.fnos.yml --env-file deploy/.env.fnos up -d --build api-server user-portal admin-portal
```

- [ ] **Step 4: 真实测试**

自动注册新测试用户，兑换/购买套餐，打开用户控制台，启动本地客户端，执行 HTTP/HTTPS/TCP/UDP 测速，确认测速隧道自动清理、流量计入套餐、页面动线完整。

## 自审

- 覆盖了用户确认的 11 项答案。
- 没有把后台和用户端合并；用户控制台仍是独立容器，本地客户端仍是独立 Win/Linux 程序。
- API Server 是测速探测发起端；本地客户端只负责 benchmark 服务与 frpc。
- 明确说明用户本地宽带限制不可消除，只能通过瓶颈说明和参考指标区分。
