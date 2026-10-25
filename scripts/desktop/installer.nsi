Unicode true
!include "MUI2.nsh"
!include "LogicLib.nsh"
Name "The War Room"
OutFile "${OUTPUT}"
InstallDir "$LOCALAPPDATA\TheWarRoom"
RequestExecutionLevel user
!define MUI_ABORTWARNING
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"
Section
  SetShellVarContext current
  InitPluginsDir
  SetOutPath "$PLUGINSDIR"
  File "${BOOTSTRAPPER}"
  ExecWait '"$PLUGINSDIR\MicrosoftEdgeWebview2Setup.exe" /silent /install' $0
  ${If} $0 != 0
    MessageBox MB_ICONSTOP "Microsoft's browser component could not be installed. Connect to the internet and try again."
    SetErrorLevel 1
    Abort
  ${EndIf}
  SetOutPath "$INSTDIR"
  File "${BINARY}"
  File /oname=INSTALL.md "${GUIDE}"
  CreateShortcut "$SMPROGRAMS\The War Room.lnk" "$INSTDIR\thewarroom.exe"
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\TheWarRoom" "DisplayName" "The War Room"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\TheWarRoom" "UninstallString" '$"$INSTDIR\Uninstall.exe$"'
SectionEnd
Section "Uninstall"
  SetShellVarContext current
  Delete "$SMPROGRAMS\The War Room.lnk"
  Delete "$INSTDIR\thewarroom.exe"
  Delete "$INSTDIR\INSTALL.md"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\TheWarRoom"
SectionEnd
