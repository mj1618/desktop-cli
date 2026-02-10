# Click --near matches wrong element when text appears multiple times

## Problem

When using `click --text "Buy groceries" --near --app "Notes"` to click a checklist checkbox in Apple Notes, the command matched and clicked on the note preview in the sidebar list (left panel) instead of the actual checklist item in the active note content area (right panel).

The command executed:
```bash
desktop-cli click --text "Buy groceries" --near --app "Notes"
```

Result:
```yaml
ok: true
action: click
x: 1115
y: 576
button: left
count: 1
```

The coordinates (1115, 576) correspond to the note preview in the middle column showing "My Task List - 9:45pm Buy groceries" rather than the actual "Buy groceries" checklist item visible in the main content area on the right side of the window.

## Proposed Fix

The `--near` text matching should prefer elements in the active/focused content area over elements in navigation/preview panes. Specifically:

1. When multiple text matches exist, prioritize elements that are:
   - In the currently focused window/pane
   - Descendants of the focused element or its ancestors
   - In larger font sizes (actual content vs. preview text)
   - Further to the right (for macOS apps with sidebar-main content layout patterns)

2. Add a `--scope-focused` or `--active-only` flag that limits text search to the active content region only, ignoring sidebar/navigation/preview elements

3. Alternatively, make the existing auto-scope feature (which scopes to dialogs/sheets) also detect and scope to the "main content area" when a multi-pane layout is detected

## Reproduction

1. Open Apple Notes with existing notes that contain checklist items
2. Create a new note with title "My Task List" and checklist items including "Buy groceries"
3. Run: `desktop-cli click --text "Buy groceries" --near --app "Notes"`
4. Observe that the click occurs on the note preview in the sidebar/list rather than on the checkbox next to "Buy groceries" in the active note content
