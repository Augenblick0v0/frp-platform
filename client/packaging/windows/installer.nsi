!ifndef VERSION
!define VERSION "0.1.0"
!endif
!ifndef APPDIR
!define APPDIR "FrpTunnelClient"
!endif
!ifndef OUTFILE
!define OUTFILE "FrpTunnelClient-${VERSION}-setup.exe"
!endif

Unicode true
Name "FRP Tunnel Client"
OutFile "${OUTFILE}"
InstallDir "$PROGRAMFILES64\FrpTunnelClient"
InstallDirRegKey HKCU "Software\FrpTunnelClient" "InstallDir"
RequestExecutionLevel admin

!include "MUI2.nsh"
!define MUI_ABORTWARNING
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOutPath "$INSTDIR"
  File /r "${APPDIR}\*"
  WriteRegStr HKCU "Software\FrpTunnelClient" "InstallDir" "$INSTDIR"
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  CreateDirectory "$SMPROGRAMS\FRP Tunnel Client"
  CreateShortcut "$SMPROGRAMS\FRP Tunnel Client\启动客户端.lnk" "$INSTDIR\start-client.bat" "" "$INSTDIR\frp-client.exe"
  CreateShortcut "$SMPROGRAMS\FRP Tunnel Client\打开 WebUI.lnk" "$INSTDIR\open-webui.bat"
  CreateShortcut "$SMPROGRAMS\FRP Tunnel Client\卸载.lnk" "$INSTDIR\Uninstall.exe"
  CreateShortcut "$DESKTOP\FRP Tunnel Client.lnk" "$INSTDIR\start-client.bat" "" "$INSTDIR\frp-client.exe"

  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FrpTunnelClient" "DisplayName" "FRP Tunnel Client"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FrpTunnelClient" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FrpTunnelClient" "Publisher" "FRP Tunnel Platform"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FrpTunnelClient" "InstallLocation" "$INSTDIR"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FrpTunnelClient" "UninstallString" "$INSTDIR\Uninstall.exe"
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FrpTunnelClient" "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FrpTunnelClient" "NoRepair" 1
SectionEnd

Section "Uninstall"
  Delete "$DESKTOP\FRP Tunnel Client.lnk"
  Delete "$SMPROGRAMS\FRP Tunnel Client\启动客户端.lnk"
  Delete "$SMPROGRAMS\FRP Tunnel Client\打开 WebUI.lnk"
  Delete "$SMPROGRAMS\FRP Tunnel Client\卸载.lnk"
  RMDir "$SMPROGRAMS\FRP Tunnel Client"
  RMDir /r "$INSTDIR"
  DeleteRegKey HKCU "Software\FrpTunnelClient"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FrpTunnelClient"
SectionEnd
