# Start snipperting, open Preferences and Editor, expand some Bulk Edit folders, save PNGs.
$ErrorActionPreference = 'Stop'
if ([Threading.Thread]::CurrentThread.GetApartmentState() -ne 'STA') {
    & powershell.exe -NoLogo -STA -File $PSCommandPath @args
    exit $LASTEXITCODE
}

Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -AssemblyName System.Drawing
Add-Type @'
using System;
using System.Runtime.InteropServices;
public class ShotWin {
    [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr hwnd, IntPtr hdcBlt, uint nFlags);
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hwnd, out RECT r);
    [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hwnd);
    public struct RECT { public int Left, Top, Right, Bottom; }
}
'@

$repo = Split-Path $PSScriptRoot -Parent
$exe = Join-Path $repo 'dist\snipperting.exe'
if (-not (Test-Path -LiteralPath $exe)) {
    $exe = Join-Path $env:LOCALAPPDATA 'snipperting\snipperting.exe'
}
if (-not (Test-Path -LiteralPath $exe)) {
    throw 'snipperting.exe not found (build or install first)'
}

if (Get-Process -Name snipperting -ErrorAction SilentlyContinue) {
    throw 'quit snipperting first'
}

$exeDir = Split-Path $exe -Parent
$dataDir = Join-Path $env:USERPROFILE '.snipperting'
$cfgPath = Join-Path $exeDir 'config.json'
if (Test-Path -LiteralPath $cfgPath) {
    $cfg = Get-Content -LiteralPath $cfgPath -Raw | ConvertFrom-Json
    if ($cfg.dataDir) { $dataDir = [string]$cfg.dataDir }
}

$editFile = $null
$note = Get-ChildItem -LiteralPath $dataDir -Recurse -Filter '*.md' -File -ErrorAction SilentlyContinue |
    Select-Object -First 1
if ($note) {
    $editFile = $note.FullName.Substring($dataDir.Length).TrimStart('\', '/')
}

$outDir = Join-Path $repo 'build\shots'
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

$startArgs = @('--prefs', '--edit')
if ($editFile) { $startArgs += $editFile }
Start-Process -FilePath $exe -ArgumentList $startArgs -WorkingDirectory $exeDir | Out-Null

$root = [System.Windows.Automation.AutomationElement]::RootElement
$scopeCh = [System.Windows.Automation.TreeScope]::Children
$scopeDesc = [System.Windows.Automation.TreeScope]::Descendants
$nameProp = [System.Windows.Automation.AutomationElement]::NameProperty

function Find-ShotWindow([string]$title) {
    $cond = New-Object System.Windows.Automation.PropertyCondition $nameProp, $title
    return $root.FindFirst($scopeCh, $cond)
}

$deadline = [datetime]::UtcNow.AddSeconds(20)
$prefs = $null
$editor = $null
while ([datetime]::UtcNow -lt $deadline) {
    if (-not $prefs) { $prefs = Find-ShotWindow 'Preferences' }
    if (-not $editor) { $editor = Find-ShotWindow 'Editor' }
    if ($prefs -and $editor) { break }
    Start-Sleep -Milliseconds 250
}
if (-not $prefs -or -not $editor) {
    throw 'Preferences or Editor did not open'
}

# WebView2 on Editor
Start-Sleep -Seconds 2

$collapsed = [char]0x25B8 + ' '
$maxExpand = 6
$expanded = 0
while ($expanded -lt $maxExpand) {
    $prefs = Find-ShotWindow 'Preferences'
    if (-not $prefs) { throw 'Preferences closed' }
    # Owner-draw folder rows are class Button but UIA ControlType Pane.
    $kids = $prefs.FindAll($scopeDesc, [System.Windows.Automation.Condition]::TrueCondition)
    $hit = $null
    foreach ($b in $kids) {
        if ($b.Current.ClassName -ne 'Button') { continue }
        $n = $b.Current.Name
        if ($n.StartsWith($collapsed)) {
            $hit = $b
            break
        }
    }
    if (-not $hit) { break }
    $pat = $hit.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
    [void]$pat.Invoke()
    $expanded++
    Start-Sleep -Milliseconds 400
}

function Save-WindowShot($el, [string]$path) {
    $hwnd = [IntPtr]$el.Current.NativeWindowHandle
    [void][ShotWin]::SetForegroundWindow($hwnd)
    Start-Sleep -Milliseconds 200
    $r = New-Object ShotWin+RECT
    if (-not [ShotWin]::GetWindowRect($hwnd, [ref]$r)) {
        throw "GetWindowRect failed for $path"
    }
    $w = $r.Right - $r.Left
    $h = $r.Bottom - $r.Top
    if ($w -lt 8 -or $h -lt 8) { throw "window too small for $path" }
    $bmp = New-Object System.Drawing.Bitmap $w, $h
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $hdc = $g.GetHdc()
    try {
        [void][ShotWin]::PrintWindow($hwnd, $hdc, 2)
    } finally {
        $g.ReleaseHdc($hdc)
        $g.Dispose()
    }
    $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
    $bmp.Dispose()
}

$prefs = Find-ShotWindow 'Preferences'
$editor = Find-ShotWindow 'Editor'
Save-WindowShot $prefs (Join-Path $outDir 'preferences.png')
Save-WindowShot $editor (Join-Path $outDir 'editor.png')
Write-Host "wrote $outDir"
