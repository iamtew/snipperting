#define MyAppName "snipperting"
#define MyAppVersion "0.0.1"

[Setup]
AppId={{A7C4E2B1-9F10-4D33-8C8A-7B91E2C4F018}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
DefaultDirName={localappdata}\snipperting
DefaultGroupName={#MyAppName}
PrivilegesRequired=lowest
OutputDir=..\dist
OutputBaseFilename=snipperting-setup
SetupIconFile=..\build\snipperting.ico
Compression=lzma
SolidCompression=yes
WizardStyle=modern
UninstallDisplayIcon={app}\snipperting.exe

[Files]
Source: "..\dist\snipperting.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\build\snipperting.ico"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\web\*"; DestDir: "{app}\web"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "..\docs\MANUAL.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\snipperting"; Filename: "{app}\snipperting.exe"; IconFilename: "{app}\snipperting.ico"
