# Client Packaging

## Windows 安装包

构建便携目录、zip 和可选 NSIS 安装包：

```bash
VERSION=0.1.5 ./client/packaging/windows/build-windows.sh
```

如果本机已安装 `makensis`，会额外输出：

```text
dist/windows/FrpTunnelClient-0.1.5-setup.exe
```

如果未安装 `makensis`，仍会输出：

```text
dist/windows/FrpTunnelClient-0.1.5-windows-amd64.zip
```

可通过 `FRPC_WINDOWS_PATH` 指定官方 `frpc.exe`：

```bash
FRPC_WINDOWS_PATH=/path/to/frpc.exe ./client/packaging/windows/build-windows.sh
# 或
FRPC_WINDOWS_URL=https://github.com/fatedier/frp/releases/download/<version>/<windows-amd64-zip> ./client/packaging/windows/build-windows.sh
```

Windows PowerShell 可使用：

```powershell
.\client\packaging\windows\build-windows.ps1 -Version 0.1.5 -FrpcWindowsPath C:\path\to\frpc.exe
```

## Linux 客户端

```bash
VERSION=0.1.5 ./client/packaging/linux/build-linux.sh
```

可通过 `FRPC_LINUX_PATH` 指定官方 `frpc`：

```bash
FRPC_LINUX_PATH=/path/to/frpc ./client/packaging/linux/build-linux.sh
# 或
FRPC_LINUX_URL=https://github.com/fatedier/frp/releases/download/<version>/<linux-amd64-tar.gz> ./client/packaging/linux/build-linux.sh
```

输出：

```text
dist/linux/FrpTunnelClient-0.1.5-linux-amd64.tar.gz
```

安装：

```bash
tar -xzf FrpTunnelClient-0.1.5-linux-amd64.tar.gz
cd frp-client
sudo ./install.sh
sudo systemctl start frp-tunnel-client
```
