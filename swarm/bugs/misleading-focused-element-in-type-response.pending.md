# Misleading focused element in `type` command response

## Problem

When using the `type` command without a `--target` or `--id` flag, the response includes a `focused` field that often shows an incorrect or unrelated element instead of the actual element that received the input.

Example - typing "Test List" in Reminders' "New List" dialog:

```bash
desktop-cli type --text "Test List" --app "Reminders"
```

Response:
```yaml
ok: true
action: type
text: Test List
focused:
    i: 9
    r: cell
    d: My Lists, Upgrade Available
    b:
        - 231
        - 529
        - 260
        - 22
```

The text was actually typed correctly into the "Name:" input field in the dialog, but the response claims focus is on the "My Lists" cell in the sidebar. This is misleading - it makes it appear that:
1. The typing went to the wrong place
2. Focus somehow jumped to an unrelated element
3. The command may have failed

## Root Cause

The `findFocusedElement` function in `helpers.go:642` walks the accessibility tree depth-first and returns the **first** element with `Focused: true`. However, macOS apps like Reminders mark **multiple** elements as focused simultaneously (parent cells, groups, and the actual input field all have `f: true`). The function returns the shallowest ancestor (e.g., `i: 9` "My Lists" cell) instead of the deepest, most specific focused element (e.g., `i: 50` the actual text input field).

Elements observed with `f: true` simultaneously in Reminders:
- `i: 9` (cell "My Lists, Upgrade Available") — returned by findFocusedElement (first match)
- `i: 14` (cell "Reminders")
- `i: 20` (cell "Test List")
- `i: 38` (cell "0 Completed")
- `i: 47` (cell "Incomplete")
- `i: 50` (input, value "First reminder updated") — **actual focused input**

## Proposed Fix

Change `findFocusedElement` in `helpers.go` to prefer the **deepest** (leaf-most) focused element when multiple elements report `Focused: true`. The most specific focused element (typically an input field) is the one that actually receives keyboard input.

Alternative approaches:
1. When multiple focused elements exist, prefer elements with role `input` over `cell`/`group`/`other`
2. Collect all focused elements and return the one with the smallest bounding box (most specific)
3. Use macOS `AXFocusedUIElement` attribute directly on the application element instead of walking the tree looking for the `focused` property

A previous attempt added pre-capture of the focused element before typing (in `typecmd.go`), which is directionally correct but doesn't fix this because the wrong element is returned both before and after typing.

## Reproduction

1. Open Reminders app: `desktop-cli open --app "Reminders"`
2. Click through welcome screens and click "Add List"
3. Type a list name: `desktop-cli type --text "Test List" --app "Reminders"`
4. Observe the response shows `focused` as an unrelated cell, even though the text was correctly typed into the name input field
5. Click OK, then type a reminder: `desktop-cli type --text "First reminder"`
6. Again observe the response shows incorrect focused element (sidebar cell) even though the reminder was successfully created
