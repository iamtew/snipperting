set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

# git tag at HEAD, or "dev" when untagged
version := `git describe --tags --exact-match 2>$null; if (-not $?) { "dev" }; exit 0`
goflags := "-ldflags \"-X main.version=" + version + "\""

# list recipes
default:
    just --list

# run Go tests (parser + editor.html contract)
test:
    go test ./...

# copy logos and build dist/snipperting.exe
build: docs-logo
    if (!(Test-Path dist)) { New-Item -ItemType Directory dist | Out-Null }
    if (!(Test-Path web)) { New-Item -ItemType Directory web | Out-Null }
    Copy-Item -Force assets/logo.png web/logo.png
    go build {{goflags}} -o dist/snipperting.exe ./cmd/snipperting

# build then start the tray app
run: build
    dist/snipperting.exe

# derive build/snipperting.ico from assets/logo.png (original stays)
icon:
    if (!(Test-Path build)) { New-Item -ItemType Directory build | Out-Null }
    go run ./tools/png2ico assets/logo.png build/snipperting.ico

# copy assets/logo.png into docs/ for GitHub Pages (original stays)
docs-logo:
    Copy-Item -Force assets/logo.png docs/logo.png

# rasterize web/vendor/tabler/*.svg to 24px ico for tray and editor chrome
tabler:
    go run ./tools/tabler

# download Monaco into web/vendor/monaco (gitignored; needed for the editor)
vendor-monaco:
    go run ./tools/vendor-monaco

# install to %LOCALAPPDATA%\snipperting and add a Start Menu shortcut
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

# zip a release; run Inno Setup if ISCC is on PATH
dist: build icon docs-logo
    if (!(Test-Path dist)) { New-Item -ItemType Directory dist | Out-Null }
    Compress-Archive -Force -Path dist/snipperting.exe,build/snipperting.ico,web,docs/MANUAL.md -DestinationPath dist/snipperting-{{version}}.zip
    if (Get-Command ISCC -ErrorAction SilentlyContinue) { ISCC /DMyAppVersion={{version}} installer/snipperting.iss }
