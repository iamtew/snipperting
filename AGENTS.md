# snipperting

Windows SysTray snippets app. Spec is the Bible: [docs/app-spec.md](docs/app-spec.md). App and spec stay in line. Changes not in the spec need a spec update first (Clanker can draft it).

- **Logo:** [assets/logo.svg](assets/logo.svg) is Meat Bag owned. Never modify it, leftover `assets/logo.png`, or `assets/logo.af`. Copy/derive only via Just (`just icon`, `just docs-logo`).
- **Stack:** Go 1.26, Just. WebView2 maps `web/` to `https://snipperting.example/` for Monaco. Tray, prefs, help, and about are native. Editor WebView2 hosts Monaco and preview HTML. Help WebView2 shows rendered MANUAL.md via NavigateToString.
- **Config:** `config.json` next to the executable, created on first start. Not shipped. Keys: `dataDir` (default `~/.snipperting`), `clipNotice` (bool, default true), `theme` (`system` / `light` / `dark`, default `system`). Native windows and Monaco follow `theme`.
- **Data:** Markdown files with optional `----` header (`label`, `about`, `action`, `state`, `hidden`). Subfolders = menu categories. Read and written natively.
- **Verify:** `just test`. After `just run`, one tray process (named mutex).

## File map

```
cmd/snipperting/main.go     process, mutex, tray
internal/config             JSON config
internal/snippet            ---- parser + data-dir tree
internal/mdhtml             goldmark HTML + Help TOC
internal/winapp             tray, native windows, Monaco child, clipboard, inject
tools/png2ico               SVG (or PNG) → .ico (does not touch assets/logo.svg)
tools/tabler                Tabler SVG → 16px .ico for the tray menu
web/                        editor shell + vendored Monaco and Tabler
docs/                       GitHub Pages + MANUAL.md (embed for Help; manual.html loads it)
installer/snipperting.iss   Inno → %LOCALAPPDATA%\snipperting
```

## Just

- `just test` — parser + editor.html + Pages manual contract
- `just build` / `just run`
- `just vendor-monaco` — Monaco into `web/vendor/monaco` (not committed)
- `just icon` / `just docs-logo`
- `just install` — copy to `%LOCALAPPDATA%\snipperting`
- `just dist` — zip + Inno if `ISCC` exists

## Do not

- Edit `assets/logo.svg`, leftover `assets/logo.png`, or `assets/logo.af`
- Write snippet paths outside the data dir
- Add auto-start, global hotkeys, or other unspec’d features
