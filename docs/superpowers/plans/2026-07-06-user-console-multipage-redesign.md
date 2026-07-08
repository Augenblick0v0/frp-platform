# User Console Multipage Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把用户端从“所有功能挤在一个页面”的静态堆叠页，重构成正常商业系统动线：公开落地页 -> 独立登录/注册页 -> 登录后控制台多页面。

**Architecture:** 用户端保持原生 HTML/CSS/JS 与 Nginx 静态 SPA，不引入前端构建链；用 `location.hash` 做轻量路由。未登录只能访问公开页、登录页、注册页；登录后进入 `/app/*` 控制台路由，侧边栏点击只渲染当前页面内容并同步 active 状态。

**Tech Stack:** Vanilla HTML, CSS, JavaScript, Nginx static SPA, existing Go API, Docker Compose fnOS deployment.

## Global Constraints

- 用户端和后台端继续保持两个独立 Docker 镜像，不能合并入口。
- 登录、注册必须是独立页面/路由，不和控制台模块挤在一起。
- 未登录访问控制台必须自动转到登录页；登录成功后才能进入系统。
- 控制台侧边栏点击必须切换页面并更新 active 状态，不能只是无效锚点。
- 套餐支付继续使用现有易支付 V1 后端接口 `/api/payments/epay/orders`。
- 不改后台根路径策略：后台端口根路径直接进后台，旧随机路径继续 404。

---

## File Structure

- Modify `apps/user-web/index.html`: 重建用户端 DOM 结构与 JS 路由；分成 landing/auth/app 三类视图；实现登录、注册、套餐、隧道、流量、设置等页面渲染函数。
- Modify `apps/user-web/style.css`: 重写布局样式；提供公开页、认证页、控制台 shell、侧边栏、页面卡片、表格和响应式移动端样式。
- No API changes required: 继续消费现有 `/api/auth/*`, `/api/user/*`, `/api/tunnels`, `/api/client/traffic`, `/api/payments/epay/orders`。
- Verify `apps/user-web/nginx.conf`: 继续 SPA fallback，支持 hash 路由和直接刷新。

---

### Task 1: Replace stacked user page with route-aware DOM

**Files:**
- Modify: `apps/user-web/index.html`

**Interfaces:**
- Consumes: Existing API endpoints and DOM root `#app`.
- Produces: Route functions `navigate(route)`, `render()`, `renderLanding()`, `renderAuth(mode)`, `renderConsole(route)`, and authenticated view routes `/app/overview`, `/app/billing`, `/app/tunnels`, `/app/traffic`, `/app/settings`.

- [ ] **Step 1: Create an app root and JS state model**

Replace the static stacked sections with one `<div id="app"></div>` and a script that stores:
```js
const state = {
  token: localStorage.getItem('token') || '',
  apiBase: localStorage.getItem('apiBase') || location.origin,
  me: null,
  subscription: null,
  traffic: null,
  tunnels: [],
  plans: [],
  activeSubscription: false,
  logLines: ['ready'],
};
```

- [ ] **Step 2: Implement route guard**

Add route logic:
```js
function currentRoute() { return location.hash.replace(/^#/, '') || '/'; }
function navigate(route) { location.hash = route; }
function normalizeRoute(route) {
  if (!route || route === '/') return '/';
  if ((route.startsWith('/app')) && !state.token) return '/login';
  if ((route === '/login' || route === '/register') && state.token) return '/app/overview';
  return route;
}
```
Expected: manual `#/app/tunnels` while logged out renders login page.

- [ ] **Step 3: Render separated public/auth/app trees**

Add render entry:
```js
async function render() {
  const route = normalizeRoute(currentRoute());
  if (route !== currentRoute()) return navigate(route);
  if (route === '/login') return renderAuth('login');
  if (route === '/register') return renderAuth('register');
  if (route.startsWith('/app')) return renderConsole(route);
  return renderLanding();
}
```
Expected: only one of landing/auth/app appears in DOM at a time.

---

### Task 2: Implement separate login and register pages

**Files:**
- Modify: `apps/user-web/index.html`

**Interfaces:**
- Consumes: `api('/api/auth/send-email-code')`, `api('/api/auth/register')`, `api('/api/auth/login')`.
- Produces: Login page route `/login`, register page route `/register`, successful login redirect `/app/overview`.

- [ ] **Step 1: Render password login page**

Create `renderAuth('login')` showing only the brand nav, login card, email/password/API base inputs, and links to register/home.

- [ ] **Step 2: Render register page**

Create `renderAuth('register')` showing only registration card, email/password/code/API base inputs, send-code button, register button, and link back to login.

- [ ] **Step 3: Wire auth actions**

Attach event handlers after rendering:
```js
$('loginBtn').onclick = async () => {
  const data = await api('/api/auth/login', { method: 'POST', body: JSON.stringify({ email: $('email').value, password: $('password').value }) });
  state.token = data.access_token;
  localStorage.setItem('token', state.token);
  navigate('/app/overview');
};
```
Expected: after login, auth DOM disappears and console DOM appears.

---

### Task 3: Split console into real pages

**Files:**
- Modify: `apps/user-web/index.html`

**Interfaces:**
- Consumes: `loadDashboardData()`, `state.subscription`, `state.traffic`, `state.tunnels`, `state.plans`.
- Produces: Sidebar page routes and active nav state.

- [ ] **Step 1: Add console shell**

Render a fixed app shell with sidebar nav entries:
```js
const navItems = [
  ['/app/overview', '概览'],
  ['/app/billing', '套餐与支付'],
  ['/app/tunnels', '隧道管理'],
  ['/app/traffic', '流量与日志'],
  ['/app/settings', '账户设置'],
];
```
Every nav link uses `data-route`, not plain section anchors.

- [ ] **Step 2: Route page body**

Use `renderConsolePage(route)` to return exactly one page body. Overview contains metrics; billing contains plan purchase and redeem; tunnels contains create form and tunnel table; traffic contains report form and log; settings contains account/API base/logout.

- [ ] **Step 3: Attach nav click handlers**

```js
document.querySelectorAll('[data-route]').forEach(el => {
  el.addEventListener('click', event => {
    event.preventDefault();
    navigate(el.dataset.route);
  });
});
```
Expected: clicking sidebar updates route, active class, heading, and body content.

---

### Task 4: Redesign CSS to feel like a normal product system

**Files:**
- Modify: `apps/user-web/style.css`

**Interfaces:**
- Consumes: class names emitted by `index.html`.
- Produces: Responsive public/auth/console layout.

- [ ] **Step 1: Add product theme tokens**

Use warm orange/ink palette, strong card shadows, and clear shell spacing via CSS variables.

- [ ] **Step 2: Style auth pages as standalone pages**

Auth page must have centered card, top nav, footer, and no visible console/sidebar elements.

- [ ] **Step 3: Style console shell**

Sidebar remains visible on desktop; on mobile nav wraps horizontally. Active item uses orange background and left rail.

- [ ] **Step 4: Style page modules**

Tables, cards, forms, logs, and disabled buttons have distinct states and do not visually merge into one huge page.

---

### Task 5: Build, deploy, and verify

**Files:**
- Verify: `deploy/docker-compose.fnos.yml`
- Modify if necessary: none expected

**Interfaces:**
- Consumes: Docker Compose services `user-portal`, `api-server`, `admin-portal`.
- Produces: deployed user portal at `http://192.168.110.56:18188/`.

- [ ] **Step 1: Run Go tests**

Run:
```bash
go test ./apps/api-server/... ./client/frp-client/...
```
Expected: all tests pass.

- [ ] **Step 2: Rebuild only user portal image**

Run:
```bash
cd deploy
docker compose -f docker-compose.fnos.yml --env-file .env.fnos up -d --build user-portal
```
Expected: `frp-fnos-user-portal` is healthy/running and API/admin services are untouched unless dependency rebuild is required.

- [ ] **Step 3: Browser verify logged-out routing**

Use Playwright:
- Open `http://192.168.110.56:18188/#/app/tunnels`
- Expected route becomes `#/login`
- Expected visible text includes `密码登录`
- Expected not visible: `创建隧道` console page.

- [ ] **Step 4: Browser verify register/login/dashboard**

Use a unique email, send code, register with dev code `123456`, login, then confirm route `#/app/overview` and console sidebar visible.

- [ ] **Step 5: Browser verify nav switching**

Click sidebar `套餐与支付`, `隧道管理`, `流量与日志`, `账户设置`.
Expected: URL hash changes, active state moves, and only the target page body is shown.

---

## Self-Review

- Spec coverage: 独立登录/注册、登录后进入、侧边栏点击修复、模块拆页、参考 mefrp 的 landing/auth/dashboard 动线、用户/后台镜像分离均覆盖。
- Placeholder scan: 无 TBD/TODO/implement later。
- Type consistency: JS route names、state 字段、API helper 和 DOM helper 在各任务一致。
