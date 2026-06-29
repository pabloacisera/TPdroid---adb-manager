Unicode true

!include "MUI2.nsh"
!include "x64.nsh"

Name "TPDroid"
OutFile "..\dist\TPDroid-Setup.exe"
InstallDir "$PROGRAMFILES64\TPDroid"
RequestExecutionLevel admin
Icon "tpdroid-icon.ico"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\TPDroid.bat"
!define MUI_FINISHPAGE_RUN_TEXT "Abrir TPDroid ahora"
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "Spanish"
!insertmacro MUI_LANGUAGE "English"

Section "TPDroid" SecApp
  SectionIn RO
  SetOutPath "$INSTDIR"

  File "..\dist\activator.exe"
  ExecWait '"$INSTDIR\activator.exe"' $0
  Delete "$INSTDIR\activator.exe"
  IntCmp $0 0 cont instalerr instalerr
  instalerr:
    MessageBox MB_ICONSTOP "Código de licencia inválido. La instalación no puede continuar.$\nAdquiera una licencia para usar TPDroid."
    Abort
  cont:

  File "..\dist\tpdroid.exe"
  File "..\installer\TPDroid.bat"
  File "..\installer\tpdroid-icon.ico"
  SetOutPath "$INSTDIR\adb-binaries\windows"
  File "..\adb-binaries\windows\adb.exe"

  CreateDirectory "$SMPROGRAMS\TPDroid"
  CreateShortCut "$SMPROGRAMS\TPDroid\TPDroid.lnk" "$INSTDIR\TPDroid.bat" "" "$INSTDIR\tpdroid-icon.ico" 0
  CreateShortCut "$SMPROGRAMS\TPDroid\Desinstalar TPDroid.lnk" "$INSTDIR\uninstall.exe" "" "$INSTDIR\tpdroid-icon.ico" 0

  WriteUninstaller "$INSTDIR\uninstall.exe"

  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TPDroid" "DisplayName" "TPDroid"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TPDroid" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TPDroid" "DisplayVersion" "1.0"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TPDroid" "Publisher" "TP Reparacion de Celulares"
SectionEnd

Section "Crear icono en el Escritorio" SecDesktop
  CreateShortCut "$DESKTOP\TPDroid.lnk" "$INSTDIR\TPDroid.bat" "" "$INSTDIR\tpdroid-icon.ico" 0
SectionEnd

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecApp} "Aplicacion principal TPDroid"
  !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} "Crear un acceso directo en el Escritorio"
!insertmacro MUI_FUNCTION_DESCRIPTION_END

Section "Uninstall"
  Delete "$INSTDIR\tpdroid.exe"
  Delete "$INSTDIR\TPDroid.bat"
  Delete "$INSTDIR\tpdroid-icon.ico"
  Delete "$INSTDIR\adb-binaries\windows\adb.exe"
  RMDir "$INSTDIR\adb-binaries\windows"
  RMDir "$INSTDIR\adb-binaries"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"

  Delete "$SMPROGRAMS\TPDroid\TPDroid.lnk"
  Delete "$SMPROGRAMS\TPDroid\Desinstalar TPDroid.lnk"
  RMDir "$SMPROGRAMS\TPDroid"

  Delete "$DESKTOP\TPDroid.lnk"

  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TPDroid"
SectionEnd
