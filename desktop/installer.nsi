Unicode true
ManifestDPIAware true
RequestExecutionLevel user
SetCompressor /SOLID lzma
!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"
!ifndef VERSION
 !define VERSION "1.0.0"
!endif
!ifndef PAYLOAD
 !define PAYLOAD "publish"
!endif
!ifndef OUT
 !define OUT "LAS-MDT-Setup-x64.exe"
!endif
Name "Lakes Area Security MDT"
OutFile "${OUT}"
InstallDir "$LOCALAPPDATA\Programs\Lakes Area Security MDT"
BrandingText "Lakes Area Security Corporation"
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\versions\${VERSION}\LAS.MDT.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Open Lakes Area Security MDT"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"
Var InstallUpdate
Var NoLaunch
Function .onInit
 SetShellVarContext current
 ${GetParameters} $R0
 ClearErrors
 ${GetOptions} $R0 "/UPDATE" $R1
 ${IfNot} ${Errors}
  StrCpy $InstallUpdate "yes"
 ${EndIf}
 ClearErrors
 ${GetOptions} $R0 "/NOLAUNCH" $R1
 ${IfNot} ${Errors}
  StrCpy $NoLaunch "yes"
 ${EndIf}
FunctionEnd
Section "LAS MDT"
 SetOutPath "$INSTDIR\versions\${VERSION}"
 File /r "${PAYLOAD}\*.*"
 WriteUninstaller "$INSTDIR\Uninstall.exe"
 CreateDirectory "$SMPROGRAMS\Lakes Area Security"
 CreateShortcut "$SMPROGRAMS\Lakes Area Security\LAS MDT.lnk" "$INSTDIR\versions\${VERSION}\LAS.MDT.exe"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\LAS-MDT" "DisplayName" "Lakes Area Security MDT"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\LAS-MDT" "DisplayVersion" "${VERSION}"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\LAS-MDT" "Publisher" "Lakes Area Security Corporation"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\LAS-MDT" "InstallLocation" "$INSTDIR"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\LAS-MDT" "DisplayIcon" "$INSTDIR\versions\${VERSION}\LAS.MDT.exe"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\LAS-MDT" "UninstallString" '"$INSTDIR\Uninstall.exe"'
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\LAS-MDT" "URLInfoAbout" "https://cad.lakesareasecurity.com"
 WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\LAS-MDT" "NoModify" 1
 WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\LAS-MDT" "NoRepair" 1
SectionEnd
Function .onInstSuccess
 ${If} $InstallUpdate == "yes"
 ${AndIf} $NoLaunch != "yes"
  Exec '"$INSTDIR\versions\${VERSION}\LAS.MDT.exe"'
 ${EndIf}
FunctionEnd
Section "Uninstall"
 SetShellVarContext current
 Delete "$SMPROGRAMS\Lakes Area Security\LAS MDT.lnk"
 RMDir "$SMPROGRAMS\Lakes Area Security"
 ; Only this installer's fixed program directory is removed. The independent
 ; %LOCALAPPDATA%\Lakes Area Security\MDT data directory is always retained.
 RMDir /r "$LOCALAPPDATA\Programs\Lakes Area Security MDT\versions"
 Delete "$LOCALAPPDATA\Programs\Lakes Area Security MDT\Uninstall.exe"
 RMDir "$LOCALAPPDATA\Programs\Lakes Area Security MDT"
 DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\LAS-MDT"
SectionEnd
