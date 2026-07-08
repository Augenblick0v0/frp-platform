# Commercial Home and Admin Console Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把用户首页改成面向商业转化的正式产品页，并把后台从所有功能堆叠的单页改成登录后可用的多页面管理控制台。

**Architecture:** 用户端继续使用现有 Vanilla HTML/CSS/JS SPA，只替换首页营销内容，不暴露内部实现动线。后台端继续独立 Docker 镜像和静态 SPA，但重构为 hash 路由：`#/login`、`#/dashboard`、`#/users`、`#/plans`、`#/redeem`、`#/tunnels`、`#/nodes`、`#/certificates`、`#/settings`、`#/logs`；每个侧栏项切换独立页面内容。

**Tech Stack:** Vanilla HTML, CSS, JavaScript, existing Go API, Nginx static hosting, Docker Compose fnOS deployment.

## Global Constraints

- 用户端和后台端保持两个独立 Docker 镜像。
- 用户首页不得再出现“Workflow/正常系统动线/不要把模块挤在一屏”等内部实施说明。
- 用户首页要商业化：强调价值、场景、套餐、稳定性、CTA，而不是解释产品开发流程。
- 后台必须改成真正可点击切换的多页面控制台，不能继续把所有功能堆在首页。
- 后台未登录只能看到登录页；登录后进入仪表盘。
- 保留现有后台 API 能力：用户、套餐、兑换码、隧道、节点、证书、设置、日志、frps/nginx 操作。
- 修改后必须部署到本机当前 Docker 服务并用浏览器验证。
- 提供一个可登录的用户端测试账号。

---

## File Structure

- Modify `apps/user-web/index.html`: 替换 landing 页最后的内部 Workflow 区块为商业化套餐/保障/CTA 区块。
- Modify `apps/user-web/style.css`: 为商业化首页新增 pricing/security/CTA 样式，复用当前主题。
- Replace `apps/admin-web/index.html`: 用后台 hash router 重构页面结构和 JS 数据加载/操作绑定。
- Replace `apps/admin-web/style.css`: 后台改为更清晰的 SaaS admin shell、独立登录页、卡片、表格、表单、日志面板与响应式侧栏。
- No API schema changes expected.

---

### Task 1: Commercialize the user landing page

**Files:**
- Modify: `apps/user-web/index.html`
- Modify: `apps/user-web/style.css`

**Interfaces:**
- Consumes: existing landing route `/`.
- Produces: public commercial homepage with no internal implementation copy.

- [ ] Replace the `landing-band` workflow section with a `trust-band` and `commercial-grid` section:
  - 服务承诺：高速节点、HTTPS/证书、用量透明、售后支持。
  - 套餐 CTA：显示高级套餐 `¥9.90 / 30 天`，按钮进入注册/登录。
  - 场景价值：NAS、远程办公、游戏联机、开发联调。

- [ ] Ensure page copy does not contain:
  - `Workflow`
  - `正常系统动线`
  - `不要把所有模块挤在一屏`
  - `访问首页了解服务`
  - `侧边栏切换具体业务页面`

---

### Task 2: Rebuild admin as authenticated multi-page console

**Files:**
- Replace: `apps/admin-web/index.html`
- Replace: `apps/admin-web/style.css`

**Interfaces:**
- Consumes: existing admin endpoints under `/api/admin/*`.
- Produces: route-aware admin pages:
  - `#/login`
  - `#/dashboard`
  - `#/users`
  - `#/plans`
  - `#/redeem`
  - `#/tunnels`
  - `#/nodes`
  - `#/certificates`
  - `#/settings`
  - `#/logs`

- [ ] Add admin state and route guard:
```js
const state = { token: localStorage.getItem('adminToken') || '', route: '', dashboard: null, users: [], plans: [], tunnels: [], nodes: [], certificates: [], operations: [], settings: null, logLines: ['ready'] };
function normalizeRoute(route) {
  if (!route || route === '/') return state.token ? '/dashboard' : '/login';
  if (route !== '/login' && !state.token) return '/login';
  if (route === '/login' && state.token) return '/dashboard';
  return adminRoutes.some(r => r.path === route) ? route : '/dashboard';
}
```

- [ ] Login page only shows admin login card and API address input.

- [ ] Dashboard page only shows metrics, recent users/tunnels, and quick actions.

- [ ] Users page only shows users table.

- [ ] Plans page only shows plans table.

- [ ] Redeem page only shows redeem-code generation form and output log.

- [ ] Tunnels page only shows tunnel table.

- [ ] Nodes page only shows node create form, nodes table, and node action buttons.

- [ ] Certificates page only shows domain/cert/nginx operations and certificate table.

- [ ] Settings page only shows system settings, mail test, and frps controls.

- [ ] Logs page only shows operation logs and runtime log.

---

### Task 3: Create a user test account

**Files:**
- No code files.

**Interfaces:**
- Consumes: `/api/auth/send-email-code`, `/api/auth/register`, `/api/auth/login` through deployed user portal.
- Produces: one user account that can log into `http://192.168.110.56:18188/#/login`.

- [ ] Register account:
  - Email: `demo@frp-platform.local`
  - Password: `Demo@2026`
  - Code: use returned `dev_code` or `123456`.

- [ ] Verify login returns access token.

---

### Task 4: Build, deploy, and verify

**Files:**
- Verify: `deploy/docker-compose.fnos.yml`

**Interfaces:**
- Produces deployed user/admin portals:
  - User: `http://192.168.110.56:18188/`
  - Admin: `http://192.168.110.56:18189/`

- [ ] Run JS syntax checks for both portals with `node --check`.
- [ ] Run Go tests: `go test ./apps/api-server/... ./client/frp-client/...`.
- [ ] Rebuild and deploy `user-portal admin-portal`.
- [ ] Browser verify user homepage no longer contains internal workflow copy.
- [ ] Browser verify admin logged-out root redirects to `#/login`.
- [ ] Browser verify admin login, then click at least dashboard/users/plans/tunnels/settings routes and confirm page heading changes.
- [ ] Commit all changes.

---

## Self-Review

- Spec coverage: 用户首页商业化、后台布局重构、用户账密、先计划后执行、部署验证均覆盖。
- Placeholder scan: 无 TBD/TODO/implement later。
- Type consistency: route paths and state fields consistent across tasks.
