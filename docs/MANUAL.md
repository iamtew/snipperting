# snipperting manual

snipperting keeps text snippets in Markdown files and serves them from the Windows tray.

## Install

Install into `%LOCALAPPDATA%\snipperting`. Start snipperting from the Start Menu shortcut.

The editor loads Monaco inside WebView2 from files next to the app. There is no HTTP server. Preferences, Help, and About are native windows. Help shows this manual rendered as markdown, with a table of contents at the top.

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

Edit opens a native window. The file tree is native (all `.md` files and folders), with indent and file/folder icons. New note, New folder, and Delete are icon buttons on that explorer. New note and New folder land in the selected folder, or next to the selected file. Delete asks Yes/No, then removes the selected note or folder (folders include their contents). Tabs keep several notes open; close a tab with the × on the tab, or close the window. Unsaved notes prompt before discard. Ctrl+S saves the active tab; there is also a Save icon on the format bar. A markdown format bar wraps the selection (bold, italic, lists, code, link, and so on); click again to unwrap. Preview opens a second tab with the note body rendered as markdown (header omitted). The text area is Monaco inside that window. Save writes the `.md` file, header included. Monaco and preview follow the Preferences theme.

## Preferences

- **Data directory** — where snippets live. Browse opens a folder picker; picking a folder saves it
- **Show copied notice** — on-screen indicator after a clipboard copy (default on)
- **Theme** — System (follow Windows), Light, or Dark. Applies immediately.
- **Bulk Edit** — rows list filenames. Inject vs Clipboard are two buttons; the pressed one is the current action. State and hidden stay on/off and shown/hidden.

## Config

`config.json` is created next to the executable on first run. It is not shipped with the installer.
