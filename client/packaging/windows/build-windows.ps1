param(
  [string]$Version = "0.1.0",
  [string]$FrpcWindowsPath = ""
)
$ErrorActionPreference = "Stop"
$Root = Resolve-Path "$PSScriptRoot\..\..\.."
$Dist = Join-Path $Root "dist\windows"
$App = Join-Path $Dist "FrpTunnelClient"
Remove-Item -Recurse -Force $Dist -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path (Join-Path $App "webui"), (Join-Path $App "config"), (Join-Path $App "logs") | Out-Null
Push-Location (Join-Path $Root "client\frp-client")
$env:GOOS="windows"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -ldflags "-s -w" -o (Join-Path $App "frp-client.exe") .
Pop-Location
Copy-Item -Recurse -Force (Join-Path $Root "apps\client-webui\*") (Join-Path $App "webui")
@'{
  "api_base": "https://api.example.com",
  "local_webui": "http://127.0.0.1:18080",
  "frpc_path": "frpc.exe"
}
'@ | Set-Content -Encoding UTF8 (Join-Path $App "config\client.example.json")
if ($FrpcWindowsPath -and (Test-Path $FrpcWindowsPath)) {
  Copy-Item $FrpcWindowsPath (Join-Path $App "frpc.exe")
} else {
  "请把 Windows 版 frpc.exe 放到本目录。下载地址：https://github.com/fatedier/frp/releases" | Set-Content -Encoding UTF8 (Join-Path $App "README-FRPC.txt")
}
'@echo off
cd /d "%~dp0"
frp-client.exe -addr 127.0.0.1:18080 -web webui -workdir "%LOCALAPPDATA%\FrpTunnelClient" -frpc "%~dp0frpc.exe"
'@ | Set-Content -Encoding ASCII (Join-Path $App "start-client.bat")
'@echo off
start http://127.0.0.1:18080
'@ | Set-Content -Encoding ASCII (Join-Path $App "open-webui.bat")
Copy-Item (Join-Path $Root "client\packaging\windows\installer.nsi") (Join-Path $Dist "installer.nsi")
Compress-Archive -Force -Path $App, (Join-Path $Dist "installer.nsi") -DestinationPath (Join-Path $Dist "FrpTunnelClient-$Version-windows-amd64.zip")
$makensis = Get-Command makensis -ErrorAction SilentlyContinue
if ($makensis) {
  & $makensis.Source "/DVERSION=$Version" "/DAPPDIR=$App" "/DOUTFILE=$(Join-Path $Dist "FrpTunnelClient-$Version-setup.exe")" (Join-Path $Dist "installer.nsi")
} else {
  Write-Host "makensis not found; generated portable zip and NSIS script only"
}
Write-Host $Dist
