# ME Frp 风格重构与 frp-panel 架构验收报告

日期：2026-07-09
分支：`codex/mefrp-layout-architecture-refactor`

## 1. 目标范围

本次按 `D:/frpbusiness/docs/superpowers/plans/2026-07-09-mefrp-layout-architecture-refactor.md` 执行，完成：

- 用户控制台、后台管理端、本地 client-webui 全量升级为 React + Vite + Ant Design。
- 三端共享 `apps/shared/frontend` 的主题、AppShell、拓扑、指标卡、状态、日志和节点操作面板。
- 页面信息架构参考 mefrp：左侧分组菜单、顶部工具栏、工作台卡片、表格页、Drawer/Steps 动线。
- 架构表达对齐 frp-panel：Master / Server(FRPS) / Client(FRPC) / Visitor。
- 修复支付通道映射、兑换码绑定套餐、套餐新增/编辑、用户套餐编辑、节点操作按钮、折叠菜单显示。
- 飞牛真实部署与全流程验收。

## 2. 架构验收

| 项 | 结果 | 证据 |
| --- | --- | --- |
| Master Control Plane | PASS | `apps/api-server` 新增 `GET /api/user/topology`、`GET /api/admin/topology`、`GET /api/admin/orders`；文档见 `docs/architecture/frp-panel-role-map.md` |
| Admin Console | PASS | `apps/admin-web` 独立 Vite 应用和独立 Nginx 容器 `frp-fnos-admin-portal` |
| User Console | PASS | `apps/user-web` 独立 Vite 应用和独立 Nginx 容器 `frp-fnos-user-portal` |
| Client(FRPC) UI | PASS | `apps/client-webui` 独立 Vite 应用，客户端打包脚本复制构建产物 |
| Server(FRPS) Node Plane | PASS | 后台 FRPS 节点页通过 Master 调 node-agent 执行状态、配置、日志、reload/restart、nginx test/reload |
| Visitor | PASS | 文档和 UI 拓扑中明确 Visitor 只访问公网入口 |

## 3. UI/排版验收

| 页面/能力 | 结果 | 截图 |
| --- | --- | --- |
| 用户入口页中文正常、蓝白工具面板风 | PASS | `D:/frpbusiness/output/playwright/user-portal-home-fixed.png` |
| 后台入口页中文正常、蓝白工具面板风 | PASS | `D:/frpbusiness/output/playwright/admin-portal-home-fixed.png` |
| 用户总览：套餐、流量、隧道、节点、拓扑、最近隧道、快捷入口 | PASS | `D:/frpbusiness/output/playwright/user-dashboard-auth.png` |
| 用户创建隧道 Steps 页面 | PASS | `D:/frpbusiness/output/playwright/user-create-tunnel-auth.png` |
| 用户套餐支付页 | PASS | `D:/frpbusiness/output/playwright/user-billing-auth.png` |
| 后台 Master 总览：指标、拓扑、订单、支付通道 | PASS | `D:/frpbusiness/output/playwright/admin-dashboard-auth.png` |
| 后台支付方式绑定入口和订单列表 | PASS | `D:/frpbusiness/output/playwright/admin-payments-auth.png` |
| 后台 FRPS 节点按钮和 NodeOperationPanel | PASS | `D:/frpbusiness/output/playwright/admin-nodes-auth.png` |
| 后台套餐新增/编辑入口 | PASS | `D:/frpbusiness/output/playwright/admin-plans-auth.png` |
| 后台兑换码选择套餐生成 | PASS | `D:/frpbusiness/output/playwright/admin-redeem-auth.png` |
| client-webui 本机状态页同风格 | PASS | `D:/frpbusiness/output/playwright/client-webui-preview.png` |
| 中文二次转义 `\uXXXX` | PASS | 已修复，截图确认中文正常；源码 `rg "\\u[0-9a-fA-F]{4}" apps/*/src apps/shared/frontend` 无结果 |
| 隧道相关错字 | PASS | 已统一改为“隧道”；源码 UI 中无旧错字残留 |

## 4. 构建与测试验收

### 本地前端构建

```text
cd D:/frpbusiness/apps/user-web && npm run build      PASS
cd D:/frpbusiness/apps/admin-web && npm run build     PASS
cd D:/frpbusiness/apps/client-webui && npm run build  PASS
```

最近一次结果：三个构建均 `✓ built`，仅有 Vite chunk size warning，不影响产物。

### 飞牛 Go 测试

```text
cd /root/frp-platform && go test ./apps/api-server/... ./client/frp-client/...
```

结果：

```text
ok   frp-platform/apps/api-server/internal/platform
ok   frp-platform/client/frp-client/internal/clientcore
```

### 飞牛标准 Docker Compose 部署

```text
docker compose -f deploy/docker-compose.fnos.yml --env-file deploy/.env.fnos up -d --build user-portal admin-portal
```

当前容器：

```text
frp-fnos-user-portal   deploy-user-portal   Up   0.0.0.0:18188->80/tcp
frp-fnos-admin-portal  deploy-admin-portal  Up   0.0.0.0:18189->80/tcp
frp-fnos-api           deploy-api-server    Up   8080/tcp
```

入口：

- 用户控制台：`http://192.168.110.56:18188`，HTTP 200。
- 后台管理端：`http://192.168.110.56:18189`，HTTP 200。

## 5. 真实全流程验收

飞牛验收摘要：`/tmp/frp-e2e-summary.json`

| 检查项 | 结果 |
| --- | --- |
| health | PASS |
| admin login | PASS |
| admin topology role | PASS |
| payment method wxpay_zg visible | PASS |
| payment config enabled | PASS |
| send code | PASS |
| register new test user | PASS |
| user login | PASS |
| active plan exists | PASS |
| admin redeem code bound plan | PASS |
| user topology safe fields | PASS |
| user redeem activates selected plan | PASS |
| create tcp tunnel | PASS |
| wxpay order created | PASS |
| admin orders include payment | PASS |

测试数据：

- 测试用户：`codex_e2e_20260709120739@example.com`
- 兑换码：`E2E-1783570060076179431-1`
- 隧道公网入口：`frp.example.com:20000`
- 支付订单：`FP141783570060124212391`
- 支付类型：`wxpay`

## 6. 支付通道验收

| 项 | 结果 |
| --- | --- |
| `wxpay_zg` 后台可见 | PASS |
| `wxpay_zg` 映射到易支付 `wxpay` | PASS |
| 支付配置 enabled | PASS |
| 用户微信支付下单成功 | PASS |
| 后台订单列表包含新订单 | PASS |

补充 API 核验结果：

```json
{
  "userTopologyLeakCount": 0,
  "paymentEnabled": true,
  "wxpayChannel": "wxpay_zg",
  "nodeCount": 1,
  "orderFound": true
}
```

## 7. 节点动作验收

通过后台节点页同一路径的 API 对真实节点执行：

| 动作 | 结果 |
| --- | --- |
| status | PASS |
| frps-config | PASS |
| frps-logs | PASS |
| frps-reload | PASS |
| nginx-test | PASS |
| nginx-reload | PASS |
| frps-restart | PASS |

详细结果文件：`D:/frpbusiness/output/playwright/node-actions-verification.json`

## 8. 安全边界验收

| 项 | 结果 |
| --- | --- |
| 用户 topology 不暴露 `agent_token` | PASS |
| 用户 topology 不暴露 `bind_token` | PASS |
| 用户 topology 不暴露支付密钥 | PASS |
| 用户 topology 不暴露 frps/admin 敏感字段 | PASS |
| 后台支付配置只显示 configured/enabled 状态和通道，不显示明文密钥 | PASS |

## 9. 计划任务完成度

| Task | 状态 |
| --- | --- |
| Task 1 领域模型和架构文档 | DONE |
| Task 2 共享 React 前端基础层 | DONE |
| Task 3 用户控制台 React 重构 + mefrp 排版 | DONE |
| Task 4 后台管理端 React 重构 + mefrp 排版 | DONE |
| Task 5 client-webui React 重构 + 同风格排版 | DONE |
| Task 6 后端 topology 和安全字段 | DONE |
| Task 7 Docker 和 compose 调整 | DONE |
| Task 8 旧文档修复和架构改写 | DONE |
| Task 9 部署前构建测试 | DONE |
| Task 10 飞牛部署、截图对比和真实验收 | DONE |

## 10. 结论

本次 mefrp 排版动线与 frp-panel 架构重构已完成。用户端、后台端、本地客户端 UI 均已工程化为 React + Vite + Ant Design；飞牛标准 Docker Compose 部署成功；支付、套餐、兑换码、隧道、节点操作、安全字段和真实支付下单流程均已通过验收。

## 8. 安全审计跟进门禁

2026-07-09 安全审计跟进已补充发布门禁：

- `/api/client/tunnels` 不再返回全局 `FRP_TOKEN` 或 `token` 字段。
- API CORS 改为 `CORS_ALLOWED_ORIGINS` 白名单。
- 内存 Store 用户/管理员 session 增加 24 小时过期。
- 生产环境缺少 `DATABASE_URL` 时启动失败；演示兑换码仅在 `ALLOW_INSECURE_DEFAULTS=true` 下启用。
- 注册输入统一校验，空密码不会触发 SQL 路径 panic。
- `ADMIN_PASSWORD` 与 `FRP_TOKEN` 拒绝弱值、占位值和短密钥。
- 本地 client-webui 与用户端测速链路均按本地 API 要求携带 `X-Local-Token`。

最终验证证据见本文件后续 `Final Security Follow-up Evidence` 章节。

## Final Security Follow-up Evidence

- Local commit: `8ad024d728469542d2c412e69af151505c80ab84`
- API tests: PASS — `ALLOW_INSECURE_DEFAULTS=true go test ./apps/api-server/...`
- Client tests: PASS — `go test ./client/frp-client/...`
- User/Admin/Client builds: PASS — `npm run build` in `apps/user-web`, `apps/admin-web`, `apps/client-webui`
- Compose config: PASS — standard, control, and fnOS compose files rendered successfully.
- Release packaging: PASS — `./scripts/release.sh 0.1.6` and `./scripts/verify-release.sh 0.1.6`
- fnOS backend hash: `c64e90f0446415a213655106d233984ff5be9d5ae0af7ea08fde0caddb55b7c4` (local `dist/fnos/api-server` and container `/app/api-server` match)
- fnOS health checks: PASS — `http://192.168.110.56:18188/health` and `http://192.168.110.56:18189/health` returned HTTP 200.
- Direct API CORS check: PASS — no `Access-Control-Allow-Origin` for empty or unconfigured evil origin.
