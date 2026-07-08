# Split Admin User Flow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 FRP 平台本机部署改成用户端和后台管理端分离入口，后台入口使用独立端口直达，并重做用户端为“未登录只显示登录/注册，登录后才进入隧道控制台”的合理动线。

**Architecture:** 当前 `deploy/nginx-fnos/nginx.conf` 把用户端、后台和 API 都堆在同一个入口下，且后台固定 `/admin/`；本次改成两个独立入口镜像：用户端镜像内置 Nginx、用户 Web 和用户 API 反代；后台镜像内置 Nginx、后台 Web、随机短路径和管理 API 反代。用户端 `apps/user-web/index.html` 改为单页状态机：`auth` 视图负责登录/注册，`app` 视图负责概览、套餐、购买、兑换、隧道配置和流量；未登录时不渲染隧道配置表单。

**Tech Stack:** Docker Compose, Nginx, Go API server, vanilla HTML/CSS/JavaScript, existing REST APIs.

## Global Constraints

- 先计划再执行修改。
- 服务端后台和用户端必须分开，不再共用一个明显的 `/admin/` 入口。
- 用户端和后台端各自使用一个 Docker 镜像/容器入口，不再用额外 sidecar Nginx 组合出入口。
- 用户端不能把全部功能堆在主页；未登录只能看到登录/注册/购买说明，登录后才能配置隧道。
- 后台使用独立端口根路径直达，不再叠加路径。
- 保留现有易支付 V1 接入、套餐购买、兑换码、隧道创建、测速和 E2E 能力。
- 不能提交真实支付密钥或运行时 `.env.fnos`。

---

## File Structure

- Modify: `deploy/docker-compose.fnos.yml`
  - Replace sidecar `user-nginx`/`admin-nginx` with two portal services: `user-portal` and `admin-portal`.
  - `user-portal` builds one user Docker image from `apps/user-web/Dockerfile` and publishes `${FNOS_USER_HTTP_PORT:-18188}:80`.
  - `admin-portal` builds one admin Docker image from `apps/admin-web/Dockerfile` and publishes `${FNOS_ADMIN_HTTP_PORT:-18189}:80`.
- Modify: `deploy/.env.example`
  - Add non-secret examples for `FNOS_USER_HTTP_PORT`, `FNOS_ADMIN_HTTP_PORT`, `FNOS_ADMIN_PATH`.
- Modify runtime-only: `deploy/.env.fnos`
  - Set actual random admin path. This file stays ignored.
- Modify/Create: `apps/user-web/nginx.conf`
  - User image serves `/` static user UI and proxies `/api/` to `api-server:8080`; `/admin/` returns 404.
- Modify/Create: `apps/admin-web/nginx.conf.template`
  - Admin image serves `/${FNOS_ADMIN_PATH}/` static admin UI and proxies `/api/` to `api-server:8080`; `/` and `/admin/` return 404.
- Modify: `apps/user-web/Dockerfile` and `apps/admin-web/Dockerfile`
  - Each Dockerfile produces a complete portal image, not just static files behind an external Nginx.
- Modify: `apps/user-web/index.html`
  - Rewrite markup into auth shell and logged-in app shell.
  - Add JS state transitions: `renderAuth()`, `renderApp()`, `requireAuth()`.
  - Move tunnel creation, traffic report, purchase and redeem controls into logged-in panels only.
- Modify: `apps/user-web/style.css`
  - Support auth landing, logged-in dashboard, panel tabs/sections, mobile layout.
- Test: existing Go tests plus deployment smoke tests.

## Task 1: Split fnOS User/Admin Entrypoints

**Files:**
- Modify: `deploy/docker-compose.fnos.yml`
- Modify: `deploy/nginx-fnos/nginx.conf` or replace with `user.conf` and `admin.conf`
- Modify runtime-only: `deploy/.env.fnos`

**Interfaces:**
- User URL: `http://192.168.110.56:${FNOS_USER_HTTP_PORT}/`
- Admin URL: `http://192.168.110.56:${FNOS_ADMIN_HTTP_PORT}/`
- API remains available to both Nginx entrypoints under `/api/`.

- [ ] Step 1: Use the dedicated admin port root path directly.
- [ ] Step 2: Build `user-portal` and `admin-portal` as the only two frontend entry containers.
- [ ] Step 3: Configure user portal image so `/admin/` and the random admin path are not reachable from the user port.
- [ ] Step 4: Configure admin portal image so `/` serves admin UI directly on the dedicated admin port.
- [ ] Step 5: Run `docker compose -f deploy/docker-compose.fnos.yml --env-file deploy/.env.fnos config`.

## Task 2: Rebuild User Flow

**Files:**
- Modify: `apps/user-web/index.html`
- Modify: `apps/user-web/style.css`

**Interfaces:**
- Unauthenticated users see login/register and a public plan teaser only.
- Authenticated users see dashboard stats, subscription, purchase/redeem, tunnel creation, tunnels table, and traffic report.
- `localStorage.token` controls initial state; logout clears token.

- [ ] Step 1: Replace always-visible shell with `<section id="authView">` and `<section id="appView" hidden>`.
- [ ] Step 2: Move API base, email, password, code, login/register controls into `authView`.
- [ ] Step 3: Move tunnel creation and traffic report into `appView` only.
- [ ] Step 4: Add `logoutBtn` and `renderAuth()/renderApp()`.
- [ ] Step 5: Keep payment order creation using `/api/payments/epay/orders`, but only after login.

## Task 3: Verify and Redeploy

**Files:**
- No new source files beyond Tasks 1-2.

**Commands:**
- `go test ./apps/api-server/... ./client/frp-client/...`
- `docker compose -f deploy/docker-compose.fnos.yml --env-file deploy/.env.fnos config`
- `docker compose -f deploy/docker-compose.fnos.yml --env-file deploy/.env.fnos up -d --build`
- Smoke checks:
  - User URL `/` returns user login page.
  - User URL `/admin/` returns 404.
  - Admin URL `/` returns admin page.
  - Admin URL `/admin/` returns 404.
  - User login works.
  - Logged-in user can list plans and create payment order.
  - Existing full FRP E2E still passes.

## Self-Review

- Requirement 1 covered by Task 1: user portal and admin portal are separate Docker images/containers, with no `/admin/` on user port.
- Requirement 2 covered by Task 2: unauthenticated auth view and authenticated app dashboard.
- Requirement 3 revised by user: dedicated admin port root path now opens admin directly, with no extra path.
- No real payment key is added to tracked files.
