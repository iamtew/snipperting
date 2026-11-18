# snipperting

Keep bits of text you reuse all the time, then fire them from the little icon by the clock. Click a snippet to copy it, or drop it straight into whatever you were typing.

## Getting started

After you install snipperting, start it from the Start Menu. A tray icon shows up near the clock (you may need to click the little arrow to see it). No terminal window opens.

Click that icon for the menu. First run also creates a snippets folder named `.snipperting` in your user folder. You can pick a different folder later in Preferences.

Only one snipperting runs at a time. Starting it again while it is already in the tray does nothing — use the icon that is already there.

## Use a snippet

Open the tray menu and click a snippet.

- **Clipboard** puts the snippet on the clipboard. Paste with Ctrl+V as usual. A small “Copied to clipboard” notice can appear; it does not steal the keyboard. Turn it off in Preferences if it gets in the way.
- **Inject** pastes into the last window you were typing in. The tray click would otherwise steal focus, so snipperting remembers that window (it skips itself and the Windows tray). Click the other app first, then the tray snippet.

Each snippet has its own action. Change it in Preferences → Bulk Edit, or in the snippet file.

**Disabled** snippets stay on the menu but are grayed out and do nothing. **Hidden** snippets are not on the menu at all. You can still see both in Edit.

## The tray menu

Left-click or right-click the icon; both open the same menu.

Snippets at this folder level are listed first, then folders. A folder opens to the side as another menu, so you can group related notes.

Hover a snippet to see its short description, if it has one.

Under the line at the bottom:

- **Edit** — write and organize snippets
- **Preferences** — folder, theme, and bulk settings
- **Help** — this guide
- **About** — version, website, and GitHub repo
- **Exit** — quit snipperting (the icon goes away until you start it again)

## Edit your snippets

Open **Edit** from the tray. The left side is a file tree of your snippets folder. Folders are listed first, then files. Names are the real file and folder names on disk (not the menu label). Hidden snippets still show up here so you can find them.

On the explorer row: **New note**, **New folder**, and **Save** on the left; **Delete** on the right. Right-click the tree for the same actions, plus **Rename**.

- New note and New folder land in the folder you selected, or in the same folder as the selected file.
- A new note is a Markdown file with a small header on top. It starts as clipboard, enabled, and visible on the menu.
- Save writes the open note (or press Ctrl+S). The floppy icon does the same.
- Rename changes the file or folder name. Notes keep a `.md` suffix if you leave it off.
- Drag a file or folder onto a folder to move it. Drop on a file puts it in that file’s folder. Drop on empty space is the snippets folder root.
- Delete asks Yes/No, then removes the selected note or folder. A folder takes everything inside it. Open tabs for those files close.

With no note open, the right side matches the window theme and tells you to open a note from the list.

You can keep several notes open as tabs. Close a tab with the × on the tab. If you have unsaved changes, you get a chance to keep them before a tab or the window goes away.

Above the text: a format bar on the left (bold, italic, strikethrough, heading, quote, lists, code, link). It wraps the selection; click the same button again to take the marks off. **Preview** sits on the right of that row and opens a second tab with the note rendered as Markdown. Preview shows the body only (not the header) and is not saved as its own file. Close the note and its preview goes with it.

## What a snippet file is

Each snippet is a `.md` file in your snippets folder. Subfolders become menu categories.

You can edit these files here in snipperting or in any text editor. They are just text.

An optional header at the top is **not** copied or injected — only the body is. If you skip the header, the file name is the menu name, the action is clipboard, and the item is enabled and visible.

```
----
label: Meeting sign-off
about: Closing line for email
action: clipboard
state: enable
hidden: false
----
Thanks for your time today. I will send notes after the call.
```

- **label** — text on the menu (the file name if you leave this blank)
- **about** — the hover hint on the menu
- **action** — `clipboard` or `inject`
- **state** — `enable` or `disable` (disabled stays on the menu, gray)
- **hidden** — `true` hides it from the tray; `false` shows it

## Preferences

- **Data directory** — where the snippet files live. Browse opens a folder picker. Pick a folder and it is saved; Bulk Edit refreshes to match. Cancel leaves the path as it was.
- **Show copied notice** — the small on-screen “copied” after a clipboard snippet (inject never shows it).
- **Theme** — System (follow Windows), Light, or Dark. It applies right away to snipperting’s windows, the editor, and Help.
- **Bulk Edit** — one row per snippet. At each indent, files come first, then folders (same as the tray). Inject and Clipboard are two buttons; the pressed one is the current action. Enabled and hidden are on/off. Changes are written into each file.
