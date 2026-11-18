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
using System.Text;
using System.Runtime.InteropServices;
public class ShotWin {
    public delegate bool EnumCB(IntPtr h, IntPtr l);
    [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr hwnd, IntPtr hdcBlt, uint nFlags);
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hwnd, out RECT r);
    [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hwnd);
    [DllImport("user32.dll")] public static extern IntPtr SendMessage(IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam);
    [DllImport("user32.dll")] public static extern bool EnumWindows(EnumCB cb, IntPtr l);
    [DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern int GetWindowText(IntPtr h, StringBuilder s, int n);
    [DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern int GetClassName(IntPtr h, StringBuilder s, int n);
    public struct RECT { public int Left, Top, Right, Bottom; }
    public static IntPtr FindFrame(string titlePrefix) {
        IntPtr found = IntPtr.Zero;
        EnumWindows((h, l) => {
            var c = new StringBuilder(256);
            GetClassName(h, c, 256);
            if (c.ToString() != "snippertingFrame") return true;
            var t = new StringBuilder(256);
            GetWindowText(h, t, 256);
            if (t.ToString().StartsWith(titlePrefix)) { found = h; return false; }
            return true;
        }, IntPtr.Zero);
        return found;
    }
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
# cwd = repo so findWebRoot sees ./web (config.json still sits next to the exe)
Start-Process -FilePath $exe -ArgumentList $startArgs -WorkingDirectory $repo | Out-Null

$scopeDesc = [System.Windows.Automation.TreeScope]::Descendants

$deadline = [datetime]::UtcNow.AddSeconds(20)
$prefsHwnd = [IntPtr]::Zero
$editorHwnd = [IntPtr]::Zero
while ([datetime]::UtcNow -lt $deadline) {
    if ($prefsHwnd -eq [IntPtr]::Zero) { $prefsHwnd = [ShotWin]::FindFrame('Preferences') }
    if ($editorHwnd -eq [IntPtr]::Zero) { $editorHwnd = [ShotWin]::FindFrame('Editor') }
    if ($prefsHwnd -ne [IntPtr]::Zero -and $editorHwnd -ne [IntPtr]::Zero) { break }
    Start-Sleep -Milliseconds 250
}
if ($prefsHwnd -eq [IntPtr]::Zero -or $editorHwnd -eq [IntPtr]::Zero) {
    throw 'Preferences or Editor did not open'
}

# WebView2 on Editor
Start-Sleep -Seconds 2

$collapsed = [char]0x25B8 + ' '
$maxExpand = 6
$expanded = 0
while ($expanded -lt $maxExpand) {
    $prefsHwnd = [ShotWin]::FindFrame('Preferences')
    if ($prefsHwnd -eq [IntPtr]::Zero) { throw 'Preferences closed' }
    $prefs = [System.Windows.Automation.AutomationElement]::FromHandle($prefsHwnd)
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
    $bh = [IntPtr]$hit.Current.NativeWindowHandle
    [void][ShotWin]::SendMessage($bh, 0x00F5, [IntPtr]::Zero, [IntPtr]::Zero)
    $expanded++
    Start-Sleep -Milliseconds 400
}

function Save-WindowShot([string]$titlePrefix, [string]$path) {
    $hwnd = [ShotWin]::FindFrame($titlePrefix)
    if ($hwnd -eq [IntPtr]::Zero) { throw "no $titlePrefix window" }
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

Save-WindowShot 'Preferences' (Join-Path $outDir 'preferences.png')
Save-WindowShot 'Editor' (Join-Path $outDir 'editor.png')
Write-Host "wrote $outDir"
