# Collapsible Sidebar and User Certificate Requests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用户端和后台端侧边选项栏支持收缩/展开；证书申请入口迁移到用户端，后台只负责查看/续期/运维管理全部证书；登录页不再显示 URL/API 地址输入框。

**Architecture:** 保持现有 Vanilla HTML/CSS/JS SPA 与独立 Docker 镜像结构。侧栏收缩状态使用 localStorage 保存，shell 增加 collapsed class；证书申请新增用户认证 API `/api/user/certificates/request`，复用现有 automation 与 certificate store；后台证书页移除申请按钮，仅保留全局证书列表、续期、Nginx 测试/重载等管理操作。

**Tech Stack:** Vanilla HTML/CSS/JS, Go net/http API, existing Automation and certificate store, Nginx static portals, Docker Compose fnOS deployment.

## Global Constraints

- 用户端和后台端继续使用独立 Docker 镜像。
- 登录/注册页不得显示 URL/API 地址输入框；API 固定使用当前站点 `location.origin`。
- 用户端侧栏和后台侧栏都要有可点击的收缩/展开按钮，并记住状态。
- 收缩时侧栏变窄，只保留品牌/按钮和菜单主标签的紧凑呈现；展开后恢复完整标签和描述。
- 证书申请由用户端发起；后台页面不再提供“申请证书”按钮。
- 后台仍可管理全部证书：查看证书列表、续期到期证书、强制续期、检测 CNAME、生成 Nginx、测试/重载 Nginx。
- 修改后必须运行 JS 语法检查、Go 测试、部署并用浏览器验证。

---

## File Structure

- Modify `apps/api-server/internal/platform/server.go`: 新增用户证书申请路由和 handler。
- Modify `apps/user-web/index.html`: 隐藏登录/注册 URL 输入，API base 内置；新增 `/app/certificates` 路由；新增用户端证书申请页面；新增侧栏收缩状态和按钮。
- Modify `apps/user-web/style.css`: 新增用户端 collapsed sidebar 样式和证书页样式。
- Modify `apps/admin-web/index.html`: 隐藏后台登录 URL 输入，API base 内置；新增后台侧栏收缩状态和按钮；证书页移除申请证书按钮。
- Modify `apps/admin-web/style.css`: 新增后台 collapsed sidebar 样式。

---

### Task 1: API endpoint for user certificate requests

**Files:**
- Modify: `apps/api-server/internal/platform/server.go`

**Interfaces:**
- Consumes: authenticated user, JSON `{ "domain": string, "email": string }`.
- Produces: `POST /api/user/certificates/request` response `{ result, record }`.

- [ ] Add route: `s.mux.HandleFunc("/api/user/certificates/request", s.auth(s.userRequestCertificate))`.
- [ ] Implement handler by reusing `s.automation.RequestCertificate`, `s.automation.CertificatePaths`, `s.automation.InspectCertificate`, and `s.store.SaveCertificate`.
- [ ] If email is empty, default it to the logged-in user email.
- [ ] Save record status as `issued`, `dry_run`, or `failed` matching admin behavior.

---

### Task 2: User portal layout changes

**Files:**
- Modify: `apps/user-web/index.html`
- Modify: `apps/user-web/style.css`

**Interfaces:**
- Consumes: `POST /api/user/certificates/request`.
- Produces: collapsible user console sidebar and `/app/certificates` page.

- [ ] Change API base to fixed `location.origin`; remove API address input from login/register/settings.
- [ ] Add nav item `['/app/certificates', '域名证书', '申请 HTTPS 证书']`.
- [ ] Add `sidebarCollapsed` state persisted in `localStorage.userSidebarCollapsed`.
- [ ] Add toggle button in user sidebar.
- [ ] Add `renderCertificatesPage()` with domain/email inputs, request button, and result log.
- [ ] Bind certificate request button to `/api/user/certificates/request`.

---

### Task 3: Admin portal layout and certificate management changes

**Files:**
- Modify: `apps/admin-web/index.html`
- Modify: `apps/admin-web/style.css`

**Interfaces:**
- Produces: collapsible admin sidebar, API base hidden, certificate page as management-only.

- [ ] Change API base to fixed `location.origin`; remove API address input from login and settings.
- [ ] Add `sidebarCollapsed` state persisted in `localStorage.adminSidebarCollapsed`.
- [ ] Add toggle button in admin sidebar.
- [ ] Remove `requestCertBtn` and “申请证书” from admin certificate page.
- [ ] Keep CNAME check, Nginx render/test/reload, renew due, force renew, and certificate table.

---

### Task 4: Build, deploy, and verify

**Files:**
- Verify: `deploy/docker-compose.fnos.yml`

**Commands:**
- `node --check /tmp/user-web-script.js`
- `node --check /tmp/admin-web-script.js`
- `go test ./apps/api-server/... ./client/frp-client/...`
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/fnos/api-server ./apps/api-server/cmd/server`
- `docker compose -f deploy/docker-compose.fnos.yml --env-file deploy/.env.fnos up -d --build api-server user-portal admin-portal`

**Browser checks:**
- User login/register pages have no API/URL input.
- User sidebar can collapse and expand.
- User certificate page exists and can call request endpoint in dry-run/current environment.
- Admin login page has no API/URL input.
- Admin sidebar can collapse and expand.
- Admin certificate page shows management controls but no “申请证书” button.

---

## Self-Review

- Spec coverage: 两端侧栏收缩、用户端证书申请、后台管理全部证书、登录 URL 内置均覆盖。
- Placeholder scan: 无 TBD/TODO/implement later。
- Type consistency: route paths, localStorage keys, endpoint paths consistent.
