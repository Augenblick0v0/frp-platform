# 验收清单

> 状态说明：`[x]` 已验证，`[ ]` 未验证，`[-]` 不适用或被后续计划替代。

## 1. 部署验收

- [x] 用户端入口 `http://192.168.110.56:18188` 返回 HTTP 200。证据：2026-07-09 `Invoke-WebRequest`。
- [x] 后台端入口 `http://192.168.110.56:18189` 返回 HTTP 200。证据：2026-07-09 `Invoke-WebRequest`。
- [x] 用户端/后台端构建 asset SHA256 与本地 `v0.1.5` 一致。
- [x] `/health` 通过前端 Nginx 代理返回 `status=ok`。
- [ ] 后端容器二进制 SHA256 与本地 `dist/fnos/api-server` 一致。

## 2. 安全验收

- [x] `/api/client/tunnels` 不返回 `FRP_TOKEN` 或 JSON 字段 `token`。证据：`TestClientTunnelsDoesNotExposeFRPToken`。
- [x] 生产环境 CORS 只允许 `CORS_ALLOWED_ORIGINS`。证据：`TestCORSRejectsUnconfiguredOrigin`。
- [x] 内存 Store 会话 24 小时后过期。证据：`TestInMemoryUserSessionExpires`、`TestInMemoryAdminSessionExpires`。
- [x] 生产环境缺少 `DATABASE_URL` 启动失败。证据：`TestRequireDatabaseURLInProduction`。
- [x] 空密码注册返回错误，不触发 panic。证据：`TestNormalizeRegistrationInputRejectsEmptyPassword`。
- [x] 弱占位密钥启动失败。证据：`TestValidateRequiredSecretsRejectsPlaceholders`。
- [x] 本地客户端 WebUI 调用本地受保护 API 自动携带 `X-Local-Token`。证据：`apps/shared/frontend/api/client.js` 与 `apps/client-webui/src/App.jsx`。
- [x] 用户端测速调用本地客户端 API 时携带 `X-Local-Token`，远端 Master API 调用不携带。证据：`apps/user-web/src/App.jsx`。

## 3. 发布前命令

- [ ] `ALLOW_INSECURE_DEFAULTS=true go test ./apps/api-server/...`
- [ ] `go test ./client/frp-client/...`
- [ ] `npm run build`：`apps/user-web`、`apps/admin-web`、`apps/client-webui`
- [ ] `docker compose config`：标准、control、fnOS compose 文件
- [ ] GitHub Release 与 tag 已发布。
