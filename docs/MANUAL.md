# snipperting manual

snipperting keeps text snippets in Markdown files and serves them from the Windows tray.

## Install

Install into `%LOCALAPPDATA%\snipperting`. Start snipperting from the Start Menu shortcut.

The app listens on `127.0.0.1` port 45690–45699 (first free port). It does not accept connections from the network.

## Data folder

Default data directory is `~/.snipperting` (`%USERPROFILE%\.snipperting`). Change it in Preferences.

Each `.md` file is one snippet. Subfolders become menu categories.

## Snippet files

Optional header (not copied or injected):

```
----
label: Demo Snippet
about: Shown when hovering the menu item
action: inject
state: enable
hidden: false
----
Snippet body goes here.
```

- `label` — menu text (filename if omitted)
- `about` — hover text
- `action` — `inject` or `clipboard` (left-click)
- `state` — `enable` / `disable` (disabled items stay visible but cannot be used)
- `hidden` — hide from the tray menu

## Tray menu

Click the tray icon for a native Windows menu (no extra window).

- Click a snippet to run its configured action (`clipboard` or `inject`)
- Disabled snippets stay in the list but are grayed out
- Folders open as submenus
- Edit, Preferences, Help, About, Exit sit under a separator
- Change action, enabled, or hidden in Preferences → Bulk Edit

**Clipboard** copies the snippet body. **Inject** pastes it at the caret in the window that was focused before the menu opened.

## Editor

Edit opens a window with a file tree of the data directory and a Markdown editor (Monaco). Save writes the `.md` file, header included.

## Preferences

- **Data directory** — where snippets live
- **Theme** — System, Light, or Dark (all snipperting web windows)
- **Bulk Edit** — change action, state, and hidden on many snippets at once

## Config

`config.json` is created next to the executable on first run. It is not shipped with the installer.
