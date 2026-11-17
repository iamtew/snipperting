# snipperting

Windows SysTray snippets app. Spec is the Bible: [docs/app-spec.md](docs/app-spec.md). App and spec stay in line. Changes not in the spec need a spec update first (Clanker can draft it).

- **Logo:** [assets/logo.png](assets/logo.png) is Meat Bag owned. Never modify it or `assets/logo.af`. Copy/derive only via Just (`just icon`, `just docs-logo`).
- **Stack:** Go 1.26, Just, stdlib HTTP on `127.0.0.1` ports **45690–45699** (first free). WebView2 windows, fyne-io/systray icon. No Wails.
- **Config:** `config.json` next to the executable, created on first start. Not shipped. Keys: `dataDir` (default `~/.snipperting`), `theme` (`system`|`light`|`dark`).
- **Data:** Markdown files with optional `----` header (`label`, `about`, `action`, `state`, `hidden`). Subfolders = menu categories.
- **Verify:** `just test`. After `just run`, `GET http://127.0.0.1:45690/api/health` (try 45690–45699).

## File map

```
cmd/snipperting/main.go     process, mutex, tray
internal/config             JSON config
internal/snippet            ---- parser + data-dir tree
internal/httpapi            pages + JSON API
internal/winapp             tray, webview, clipboard, inject
tools/png2ico               PNG → .ico (copies, does not touch assets/logo.png)
web/                        menu, editor, prefs, help, about
docs/                       GitHub Pages + MANUAL.md (also go:embed for Help)
installer/snipperting.iss   Inno → %LOCALAPPDATA%\snipperting
```

## Just

- `just test` — parser + httptest API
- `just build` / `just run`
- `just vendor-monaco` — Monaco into `web/vendor/monaco` (not committed)
- `just icon` / `just docs-logo`
- `just install` — copy to `%LOCALAPPDATA%\snipperting`
- `just dist` — zip + Inno if `ISCC` exists

## Do not

- Edit `assets/logo.png` or `assets/logo.af`
- Bind HTTP off localhost
- Write snippet paths outside the data dir
- Add auto-start, global hotkeys, or other unspec’d features
