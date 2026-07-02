# frps 管理说明

API Server 提供 frps 管理接口：

```text
GET  /api/admin/frps/status
GET  /api/admin/frps/config
GET  /api/admin/frps/logs
POST /api/admin/frps/restart
POST /api/admin/frps/reload
```

默认部署中 API Server 会只读挂载 frps 配置：

```text
./frps:/app/runtime/frps:ro
```

并挂载 frps 日志：

```text
./logs/frps:/app/runtime/logs/frps
```

可通过 `.env` 配置命令：

```env
FRPS_CONFIG_PATH=/app/runtime/frps/frps.toml
FRPS_LOG_PATH=/app/runtime/logs/frps/frps.log
FRPS_STATUS_CMD=
FRPS_RESTART_CMD=
FRPS_RELOAD_CMD=
```

默认命令留空时，后台只读取配置和日志，不直接控制容器。生产环境可接入受限的运维脚本或 sidecar，例如：

```env
FRPS_STATUS_CMD=/app/bin/frps-status
FRPS_RESTART_CMD=/app/bin/frps-restart
FRPS_RELOAD_CMD=/app/bin/frps-reload
```
