param(
  [string]$Version = "0.1.4",
  [string]$FrpcWindowsPath = ""
)
$ErrorActionPreference = "Stop"
$Root = Resolve-Path "$PSScriptRoot\..\..\.."
$Dist = Join-Path $Root "dist\windows"
$App = Join-Path $Dist "FrpTunnelClient"
Remove-Item -Recurse -Force $Dist -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path (Join-Path $App "webui"), (Join-Path $App "config"), (Join-Path $App "logs") | Out-Null

Push-Location (Join-Path $Root "apps\client-webui")
npm install
npm run build
Pop-Location

Push-Location (Join-Path $Root "client\frp-client")
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -ldflags "-s -w" -o (Join-Path $App "frp-client.exe") .
Pop-Location

Copy-Item -Recurse -Force (Join-Path $Root "apps\client-webui\dist\*") (Join-Path $App "webui")
$json = @"
{
  "api_base": "https://api.example.com",
  "local_webui": "http://127.0.0.1:18080",
  "frpc_path": "frpc.exe"
}
"@
$json | Set-Content -Encoding UTF8 (Join-Path $App "config\client.example.json")
if ($FrpcWindowsPath -and (Test-Path $FrpcWindowsPath)) {
  Copy-Item $FrpcWindowsPath (Join-Path $App "frpc.exe")
} else {
  "Please place frpc.exe next to frp-client.exe, or provide -FrpcWindowsPath." | Set-Content -Encoding UTF8 (Join-Path $App "README-FRPC.txt")
}
$startBat = @"
@echo off
cd /d "%~dp0"
frp-client.exe -addr 127.0.0.1:18080 -web webui -workdir "%LOCALAPPDATA%\FrpTunnelClient" -frpc "%~dp0frpc.exe"
"@
$startBat | Set-Content -Encoding ASCII (Join-Path $App "start-client.bat")
$openBat = @"
@echo off
start http://127.0.0.1:18080
"@
$openBat | Set-Content -Encoding ASCII (Join-Path $App "open-webui.bat")
Copy-Item (Join-Path $Root "client\packaging\windows\installer.nsi") (Join-Path $Dist "installer.nsi")
Compress-Archive -Force -Path $App, (Join-Path $Dist "installer.nsi") -DestinationPath (Join-Path $Dist "FrpTunnelClient-$Version-windows-amd64.zip")
$makensis = Get-Command makensis -ErrorAction SilentlyContinue
if ($makensis) {
  $OutFile = Join-Path $Dist "FrpTunnelClient-$Version-setup.exe"
  & $makensis.Source "/DVERSION=$Version" "/DAPPDIR=$App" "/DOUTFILE=$OutFile" (Join-Path $Dist "installer.nsi")
} else {
  Write-Host "makensis not found; generated portable zip and NSIS script only"
}
Write-Host $Dist
