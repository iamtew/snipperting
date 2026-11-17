set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

version := "0.0.1"
goflags := "-ldflags \"-X main.version=" + version + "\""

default:
    just --list

test:
    go test ./...

build: docs-logo
    if (!(Test-Path dist)) { New-Item -ItemType Directory dist | Out-Null }
    if (!(Test-Path web)) { New-Item -ItemType Directory web | Out-Null }
    Copy-Item -Force assets/logo.png web/logo.png
    go build {{goflags}} -o dist/snipperting.exe ./cmd/snipperting

run: build
    dist/snipperting.exe

icon:
    if (!(Test-Path build)) { New-Item -ItemType Directory build | Out-Null }
    go run ./tools/png2ico assets/logo.png build/snipperting.ico

docs-logo:
    Copy-Item -Force assets/logo.png docs/logo.png

vendor-monaco:
    go run ./tools/vendor-monaco

install: build icon
    $dest = Join-Path $env:LOCALAPPDATA "snipperting"
    New-Item -ItemType Directory -Force $dest | Out-Null
    Copy-Item -Force dist/snipperting.exe $dest
    Copy-Item -Force build/snipperting.ico $dest
    Copy-Item -Recurse -Force web $dest
    $programs = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs"
    $ws = New-Object -ComObject WScript.Shell
    $sc = $ws.CreateShortcut((Join-Path $programs "snipperting.lnk"))
    $sc.TargetPath = Join-Path $dest "snipperting.exe"
    $sc.IconLocation = Join-Path $dest "snipperting.ico"
    $sc.WorkingDirectory = $dest
    $sc.Save()

dist: build icon docs-logo
    if (!(Test-Path dist)) { New-Item -ItemType Directory dist | Out-Null }
    Compress-Archive -Force -Path dist/snipperting.exe,build/snipperting.ico,web,docs/MANUAL.md -DestinationPath dist/snipperting-{{version}}.zip
    if (Get-Command ISCC -ErrorAction SilentlyContinue) { ISCC installer/snipperting.iss }
