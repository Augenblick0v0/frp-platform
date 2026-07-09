# Security Notes

## frps token model

当前 frps 使用原生全局 token。平台已经禁止空值和 `change-me`，部署模板也要求通过 `FRP_TOKEN` 注入强随机值，但全局 token 仍会出现在本地 frpc 配置中。

运维要求：

- 使用至少 32 字节随机 `FRP_TOKEN`。
- 定期轮换 `FRP_TOKEN`，轮换后同步重启 frps/frpc。
- 不要把示例 `.env` 的占位值直接用于生产。
- 下一阶段应结合 frps 鉴权插件或独立准入网关实现“每用户凭证”，否则不能宣称平台完全阻止用户绕过官方客户端直接连接 frps。

## Release Gate

生产发布必须满足：

1. `/api/client/tunnels` 不返回全局 `FRP_TOKEN`。
2. `CORS_ALLOWED_ORIGINS` 已配置且不使用 `*`。
3. `DATABASE_URL` 已配置；除显式 `ALLOW_INSECURE_DEFAULTS=true` 外不得使用内存 Store。
4. 所有本地客户端写操作均要求 `X-Local-Token`。
5. 验收清单中安全验收项全部为 `[x]`。
