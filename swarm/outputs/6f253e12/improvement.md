# screenshot/focus/list fail for apps whose windows aren't in CGWindowListCopyWindowInfo

## Problem

Some apps (e.g. Apple Notes) have accessible windows that `read --app` can interact with via the accessibility tree, but `list --windows`, `screenshot --app`, and `focus --app` all fail with "no windows found". This happens because these commands enumerate windows via a different mechanism (likely `CGWindowListCopyWindowInfo`) that doesn't return windows for certain apps.

Concrete example — Notes is running and `read` works fine:

```
$ desktop-cli list --apps
- app: Notes
  pid: 61946

$ desktop-cli read --app "Notes" --format agent
# Notes
[3] btn "New Folder" (789,987,96,18) val="0"
[2110] btn "New Note" (1278,355,39,52)
... (full UI tree returned successfully)

$ desktop-cli click --text "New Note" --roles "btn" --app "Notes"
ok: true
action: click
```

But window-dependent commands all fail:

```
$ desktop-cli list --app "Notes"
[]

$ desktop-cli list --windows
(Notes not present in output at all)

$ desktop-cli screenshot --app "Notes" --output /tmp/notes.png
Error: no windows found matching the specified criteria

$ desktop-cli focus --app "Notes"
Error: no windows found for app "Notes"
```

## Proposed Fix

The window enumeration used by `list --windows`, `screenshot`, and `focus` should fall back to the accessibility tree when `CGWindowListCopyWindowInfo` (or equivalent) doesn't return a window for the app. Specifically:

1. When `--app` is provided and no CG windows are found, query the accessibility API for the app's `AXWindows` attribute to get the window reference and bounds.
2. For `screenshot --app`, if no CG window is found but an AX window exists, use the AX window's bounds to capture that screen region (or use `CGWindowListCreateImage` with the AX-derived window ID if available via `_AXUIElementGetWindow`).
3. For `focus --app`, use `AXUIElementPerformAction(kAXRaiseAction)` on the AX window or `NSRunningApplication.activate()` on the process.
4. For `list --windows`, include windows discovered via the accessibility tree that are missing from the CG window list.

This would make the tool consistent — if `read --app` can find and interact with a window, then `screenshot`, `focus`, and `list` should also work for that app.

## Reproduction

1. Open Apple Notes (`open -a Notes`)
2. Run `desktop-cli list --apps` — Notes appears
3. Run `desktop-cli read --app "Notes" --format agent` — returns full UI tree, works fine
4. Run `desktop-cli list --app "Notes"` — returns `[]`
5. Run `desktop-cli screenshot --app "Notes" --output /tmp/notes.png` — fails with "no windows found"
6. Run `desktop-cli focus --app "Notes"` — fails with "no windows found for app"
