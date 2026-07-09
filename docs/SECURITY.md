# Security Notes

## frps token model

当前 frps 使用原生全局 token。平台已经禁止空值和 `change-me`，部署模板也要求通过 `FRP_TOKEN` 注入强随机值，但全局 token 仍会出现在本地 frpc 配置中。

运维要求：

- 使用至少 32 字节随机 `FRP_TOKEN`。
- 定期轮换 `FRP_TOKEN`，轮换后同步重启 frps/frpc。
- 不要把示例 `.env` 的占位值直接用于生产。
- 下一阶段应结合 frps 鉴权插件或独立准入网关实现“每用户凭证”，否则不能宣称平台完全阻止用户绕过官方客户端直接连接 frps。
