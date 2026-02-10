# Cross-App Element Search (`find` command)

## Problem

When an agent performs an action that triggers a dialog, notification, or new window, it often doesn't know which app or window now contains the element it needs. The current workflow requires:

1. `list --windows` to see all windows
2. Guess which app/window owns the dialog
3. `read --app <name> --text "..." --flat` to search that app
4. If not found, try the next app

This is 3-4 round-trips just to find where an element appeared. Common scenarios:
- A "Save As" dialog appeared after pressing Cmd+S — is it a system dialog or app-owned?
- A notification popped up — which app produced it?
- A permission prompt appeared — is it in the app or in System Settings?
- An "Open With" chooser appeared
- A print dialog appeared
- A confirmation dialog appeared in a different window

## Proposed Solution

Add a `find` command that searches across all windows (or all windows of the frontmost app) for matching elements:

```bash
# Find an element by text across all windows
desktop-cli find --text "Save As"

# Find with role filter
desktop-cli find --text "Allow" --roles "btn"

# Find across all windows of a specific app
desktop-cli find --text "Save" --app "Safari"

# Output: includes the app name, window title, and matching elements
```

### Output Format

```yaml
ok: true
matches:
  - app: "Safari"
    window: "Untitled"
    pid: 1234
    elements:
      - i: 42
        r: btn
        t: "Save As..."
        b: [300, 400, 100, 32]
  - app: "System Preferences"
    window: "Security"
    pid: 5678
    elements:
      - i: 7
        r: btn
        t: "Allow"
        b: [500, 300, 80, 28]
```

### Agent format output

```
# Safari - Untitled (pid: 1234)
[42] btn "Save As..." (300,400,100,32)

# System Preferences - Security (pid: 5678)
[7] btn "Allow" (500,300,80,28)
```

## Implementation Notes

1. Use `ListWindows()` to enumerate all windows
2. For each window, do a text search (reuse existing `ReadElements` + text filter logic)
3. Return results grouped by app/window
4. Add `--format agent` support for compact output
5. Add `--limit` flag to cap total results (default 10) for token efficiency
6. Search frontmost window first for faster results in the common case
7. Skip minimized/hidden windows by default

## Scope

- New `cmd/find.go` file with cobra command
- Reuse existing `helpers.go` text matching and element filtering
- No new platform API calls needed — just iterates over windows using existing `ReadElements`
- Add to README.md and SKILL.md

## Value

Eliminates 2-3 round-trips when agents need to locate elements after cross-app interactions (dialogs, notifications, permission prompts). This is one of the most common sources of wasted tokens in multi-app workflows.
