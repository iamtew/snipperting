# snipperting manual

snipperting keeps text snippets in Markdown files and serves them from the Windows tray.

## Install

Install into `%LOCALAPPDATA%\snipperting`. Start snipperting from the Start Menu shortcut.

The app listens on `127.0.0.1` port 45690–45699 (first free port) so the editor can load Monaco. It does not accept connections from the network. Preferences, help, and about are native windows.

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

**Clipboard** copies the snippet body and, by default, shows a brief “Copied to clipboard” notice (does not steal focus). Turn the notice off in Preferences. **Inject** pastes at the caret in the last window you were typing in (not snipperting and not the tray), even after the tray click steals focus.

## Editor

Edit opens a native window. The file tree is native (all `.md` files and folders), with indent and file/folder icons. New note, New folder, and Save are icon buttons on that explorer. Tabs keep several notes open; close a tab with the × on the tab, or close the window. Unsaved notes prompt before discard. Ctrl+S still saves the active tab. A markdown format bar wraps the selection (bold, italic, lists, code, link, and so on). The text area is Monaco inside that window. Save writes the `.md` file, header included. Monaco follows the Preferences theme.

## Preferences

- **Data directory** — where snippets live
- **Show copied notice** — on-screen indicator after a clipboard copy (default on)
- **Theme** — System (follow Windows), Light, or Dark. Applies immediately.
- **Bulk Edit** — Inject vs Clipboard are two buttons; the pressed one is the current action. State and hidden stay on/off and shown/hidden.

## Config

`config.json` is created next to the executable on first run. It is not shipped with the installer.
