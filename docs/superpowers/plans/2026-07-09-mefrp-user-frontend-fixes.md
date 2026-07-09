# ME Frp User Frontend Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复用户端前端体验：删除面向内部架构的角色拓扑，按 ME Frp 控制台习惯简化隧道创建流程，优化测速页，并用真实 mefrp dashboard 观察结果重新调整前端信息架构与视觉样式。

**Architecture:** 只改用户端前端与共享样式，不改后端接口。用户端首页从“架构解释”改成“用户任务工作台”；创建隧道从竖向多步骤改为一页表单 + 可折叠高级设置 + 明确上一步/下一步；测速页从技术流水线改为配置、执行、结果三段式。保留后台/本地客户端现有结构，避免扩大范围。

**Tech Stack:** React 19, Vite, Ant Design 5, CSS, Playwright CLI/browser observation.

## Global Constraints

- 默认文档使用中文；代码、命令、API 字段保持英文。
- 先观察 `https://www.mefrp.com/dashboard/home` 的布局与交互，再改本地前端。
- 用户端不得展示 Master / Server / Client / Visitor 角色拓扑和内部架构流程。
- 新建隧道必须支持返回上一步，不得强迫用户按复杂架构理解流程。
- 测速页必须减少术语噪音，保留必要 token、安全字段。
- 修改后必须运行 `npm run build`，并用浏览器截图检查首页、创建隧道、测速页。

---

## File Structure

- `apps/user-web/src/App.jsx`：重写用户首页、创建隧道、测速页组件；删除 `RoleTopology` 引用。
- `apps/user-web/src/styles.css`：新增 ME Frp 风格工作台、快捷卡片、一页表单、测速结果样式。
- `apps/shared/frontend/components/AppShell.jsx`：如需调整左侧菜单体验，仅做兼容性小改，不破坏后台/客户端。
- `docs/superpowers/plans/2026-07-09-mefrp-user-frontend-fixes.md`：本计划和执行勾选。
- `output/playwright/`：保存 mefrp 参考截图和本地验证截图。

## Task 1: 观察 ME Frp 控制台并固化设计要点

**Files:**
- Evidence: `output/playwright/mefrp-dashboard-home.png`
- Modify: `docs/superpowers/plans/2026-07-09-mefrp-user-frontend-fixes.md`

**Interfaces:**
- Produces: 设计要点清单：左侧菜单、首页卡片、常用操作、列表/表单密度。

- [ ] **Step 1: 打开 mefrp 页面并使用访问密钥进入**

Run browser automation, input access key `007MPZIK015CX625015U7RWL00E37BH400FYHBQK` where required, click dashboard side navigation items.

- [ ] **Step 2: 截图与记录布局要点**

记录：左侧可切换菜单、首页更偏“资源与操作摘要”、创建隧道入口直接、没有角色拓扑解释。

- [ ] **Step 3: 提交计划文档**

```bash
git add docs/superpowers/plans/2026-07-09-mefrp-user-frontend-fixes.md
git commit -m "docs: plan mefrp user frontend fixes"
```

## Task 2: 用户首页删除角色拓扑，改成任务工作台

**Files:**
- Modify: `apps/user-web/src/App.jsx`
- Modify: `apps/user-web/src/styles.css`

**Interfaces:**
- Removes: `RoleTopology` import and usage from user console.
- Produces: `Overview` showing user-facing cards: 套餐、流量、隧道、在线节点、快捷操作、最近隧道、公告/客户端提示。

- [ ] **Step 1: 删除 RoleTopology 引用**

Remove import:
```jsx
import { RoleTopology } from '../../shared/frontend/components/RoleTopology.jsx';
```
Remove `<RoleTopology mode="user" data={state.topology} />` from `Overview`.

- [ ] **Step 2: 重写 Overview**

实现 ME Frp 风格工作台：顶部欢迎卡片、4 个数据卡、快捷操作、最近隧道、当前套餐提示；只展示用户能行动的信息。

- [ ] **Step 3: 增加样式**

新增 `.user-hero`, `.quick-action-grid`, `.compact-tunnel-list`, `.notice-strip` 等样式。

- [ ] **Step 4: 构建验证**

```bash
cd apps/user-web && npm run build
```

- [ ] **Step 5: 提交**

```bash
git add apps/user-web/src/App.jsx apps/user-web/src/styles.css
git commit -m "fix: replace user topology with dashboard workspace"
```

## Task 3: 简化新建隧道流程并支持上一步

**Files:**
- Modify: `apps/user-web/src/App.jsx`
- Modify: `apps/user-web/src/styles.css`

**Interfaces:**
- Produces: `CreateTunnel` with protocol selector, one main form, optional advanced block, footer buttons `上一步` / `下一步` / `创建隧道`.
- Preserves API: `POST /api/tunnels` payload fields unchanged.

- [ ] **Step 1: 改为两阶段轻流程**

Stage 0：选择协议 + 节点 + 本地服务。Stage 1：公网入口/限速/确认。任何时候可点“上一步”。

- [ ] **Step 2: 为 TCP/UDP/HTTP/HTTPS 给用户文案**

协议卡片用用途文案：TCP=远程桌面/SSH，UDP=游戏/语音，HTTP=网站，HTTPS=安全网站。

- [ ] **Step 3: 保持表单字段兼容后端**

提交 payload 仍为 `{ name, type, node_id, local_host, local_port, domain, bandwidth_limit_kbps }`。

- [ ] **Step 4: 构建并提交**

```bash
cd apps/user-web && npm run build
cd ../..
git add apps/user-web/src/App.jsx apps/user-web/src/styles.css
git commit -m "fix: simplify tunnel creation flow"
```

## Task 4: 优化隧道测速页

**Files:**
- Modify: `apps/user-web/src/App.jsx`
- Modify: `apps/user-web/src/styles.css`

**Interfaces:**
- Produces: `SpeedTest` with local client connection card, test settings card, run progress, readable result summary.
- Local API calls continue carrying `X-Local-Token`.
- Remote Master API calls continue without `X-Local-Token`.

- [ ] **Step 1: 重排测速表单**

把本地客户端地址/token 放在“本地客户端连接”卡片；把节点、协议、下载/上传大小放在“测速参数”卡片。

- [ ] **Step 2: 简化进度文案**

把技术步骤改为：准备本地测速服务、创建临时入口、同步配置、执行测速、清理临时资源。

- [ ] **Step 3: 解析结果摘要**

从返回 JSON 中提取 `metrics.download_average_kbps`、`metrics.upload_average_kbps`，显示 Mbps 卡片；原始日志折叠保留。

- [ ] **Step 4: 构建并提交**

```bash
cd apps/user-web && npm run build
cd ../..
git add apps/user-web/src/App.jsx apps/user-web/src/styles.css
git commit -m "fix: streamline tunnel speed test page"
```

## Task 5: 浏览器验证和部署准备

**Files:**
- Evidence: `output/playwright/user-home-mefrp-fix.png`
- Evidence: `output/playwright/user-create-mefrp-fix.png`
- Evidence: `output/playwright/user-speed-mefrp-fix.png`

**Interfaces:**
- Verifies: 左侧菜单切换、首页无角色拓扑、创建隧道可上一步、测速页可填写 local token。

- [ ] **Step 1: 本地启动 user-web preview**

```bash
cd apps/user-web && npm run build && npm run preview -- --host 127.0.0.1 --port 4176
```

- [ ] **Step 2: Playwright 截图检查**

访问 overview/create/speedtest 三页，保存截图到 `output/playwright/`。

- [ ] **Step 3: 最终提交验证记录**

```bash
git status --short
git log --oneline -5
```

## Self-Review

- Spec coverage: 覆盖删除角色拓扑、简化隧道创建并支持上一步、优化测速、参考 mefrp 左侧切换与首页模板。
- Placeholder scan: 无 TBD/TODO/implement later。
- Type consistency: 保持现有 `api.post('/api/tunnels')` 和测速 API 字段不变。
