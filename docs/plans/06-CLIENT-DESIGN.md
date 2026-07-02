# 客户端设计

## 1. 客户端目标

客户端负责用户登录、套餐展示、隧道创建、frpc 进程管理、日志展示和状态上报。

客户端支持：

- Linux WebUI。
- Windows 安装包。

## 2. 客户端架构

```text
frp-client
├── core
│   ├── auth
│   ├── api
│   ├── tunnel
│   ├── frpc-manager
│   ├── log-manager
│   └── config-manager
├── webui
├── embedded-frpc
└── package
```

## 3. Linux 客户端

目录：

```text
/opt/frp-client/
├── frp-client
├── frpc
├── webui/
├── config/
└── logs/
```

功能：

- 启动本地 WebUI。
- 登录账号。
- 保存 Token。
- 创建隧道。
- 拉取隧道配置。
- 启动 frpc。
- 停止 frpc。
- 查看日志。
- 安装 systemd 服务。

## 4. Windows 客户端

安装目录：

```text
C:\Program Files\FrpTunnelClient\
├── frp-client.exe
├── frpc.exe
├── webui\
├── config\
└── logs\
```

安装包要求：

- 创建开始菜单快捷方式。
- 创建卸载程序。
- 可选创建桌面快捷方式。
- 可选开机自启。

## 5. 本地 WebUI 页面

页面：

- 登录页。
- 注册页。
- 验证码页。
- 套餐状态页。
- 兑换码页。
- 隧道列表页。
- 新增隧道页。
- 日志页。
- 设置页。

## 5.1 WebUI 视觉风格

Linux WebUI 和 Windows 内置 WebUI 采用 Cloudflare 网页风格。

设计要求：

- 白色/浅灰背景。
- 橙色按钮和重点状态。
- 隧道列表使用卡片加表格组合。
- 日志页使用深色等宽字体区域。
- 新增隧道流程要像 Cloudflare 创建规则一样清晰分步。
- 错误状态直接显示可操作建议，例如 CNAME 未生效、套餐不支持 HTTPS、证书申请失败。

## 6. 创建隧道表单

TCP/UDP：

```text
隧道名称
协议类型
本地地址
本地端口
```

HTTP/HTTPS：

```text
隧道名称
协议类型
本地地址
本地端口
绑定域名
是否开启 HTTPS
```

## 7. frpc 配置生成

客户端从服务端拉取 JSON 配置后生成本地 frpc 配置。

示例：

```toml
serverAddr = "frp.example.com"
serverPort = 7000
auth.method = "token"
auth.token = "runtime-token"

[[proxies]]
name = "web-1001"
type = "http"
localIP = "127.0.0.1"
localPort = 8080
customDomains = ["app.user.com"]
```

## 8. 日志

客户端需要采集：

- 客户端自身日志。
- frpc stdout/stderr。
- API 请求失败日志。
- 隧道启动/停止日志。

日志展示按时间倒序，支持刷新和复制。
