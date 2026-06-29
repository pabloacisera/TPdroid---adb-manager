[Setup]
AppName=TPDroid
AppVersion=1.0
AppPublisher=TP Reparacion de Celulares
AppPublisherURL=
DefaultDirName={autopf}\TPDroid
DefaultGroupName=TPDroid
OutputDir=..\dist
OutputBaseFilename=TPDroid-Setup
Compression=lzma
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
SetupIconFile=..\installer\tpdroid-icon.ico

[Languages]
Name: "spanish"; MessagesFile: "compiler:Languages\Spanish.isl"

[Tasks]
Name: "desktopicon"; Description: "Crear icono en el Escritorio"; GroupDescription: "Iconos adicionales:"; Flags: checked

[Files]
Source: "..\dist\tpdroid.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\adb-binaries\windows\adb.exe"; DestDir: "{app}\adb-binaries\windows"; Flags: ignoreversion
Source: "..\installer\TPDroid.bat"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\installer\tpdroid-icon.ico"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autodesktop}\TPDroid"; Filename: "{app}\TPDroid.bat"; WorkingDir: "{app}"; IconFilename: "{app}\tpdroid-icon.ico"; Tasks: desktopicon; Comment: "Abrir TPDroid"
Name: "{group}\TPDroid"; Filename: "{app}\TPDroid.bat"; WorkingDir: "{app}"; IconFilename: "{app}\tpdroid-icon.ico"; Comment: "Abrir TPDroid"
Name: "{group}\Desinstalar TPDroid"; Filename: "{uninstallexe}"; IconFilename: "{app}\tpdroid-icon.ico"

[Run]
Filename: "{app}\TPDroid.bat"; Description: "Abrir TPDroid ahora"; Flags: postinstall nowait skipifsilent shellexec
