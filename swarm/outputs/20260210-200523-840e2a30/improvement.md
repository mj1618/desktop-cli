# Menu bar not visible in app-scoped screenshots

## Problem

When taking screenshots with `--app` flag, the macOS menu bar (containing File, Edit, Format, View, Window, Help) is not captured in the output. This makes it impossible to visually locate menu items for coordinate-based clicking.

Example command:
```bash
desktop-cli screenshot --app "TextEdit" --output /tmp/textedit.png
```

The resulting screenshot only shows the window content, not the menu bar at the top of the screen. This is problematic because:
1. Menu items are not exposed in the accessibility tree (`read --format agent` or `read --roles "menu,menuitem"` returns no results for the menu bar)
2. Without seeing the menu bar in screenshots, users cannot determine coordinates for menu clicks
3. This forces reliance on keyboard shortcuts (if known) or full-screen screenshots (which include other apps/desktop)

## Proposed Fix

Add a `--include-menubar` flag to the `screenshot` command that, when combined with `--app`, captures both the focused window AND the macOS menu bar at the top of the screen.

```bash
desktop-cli screenshot --app "TextEdit" --include-menubar --output /tmp/textedit-with-menu.png
```

This would create a composite image showing:
- The menu bar at the top (with the app's menus visible: File, Edit, Format, etc.)
- The app window below it

The coordinates in the screenshot should remain accurate for clicking purposes.

Alternative implementation: Make menu bar inclusion automatic when `--app` is specified, since it's part of the app's UI in macOS.

## Reproduction

1. Open TextEdit: `open -a "TextEdit"`
2. Create a new document
3. Take screenshot: `desktop-cli screenshot --app "TextEdit" --output /tmp/test.png`
4. Observe that the menu bar (File, Edit, Format, View, Window, Help) is not visible in the captured image
5. Try to read menu items: `desktop-cli read --app "TextEdit" --roles "menu,menuitem" --format agent` — returns empty
6. This makes menu-based interactions require either:
   - Blind coordinate clicking (error-prone)
   - Full screen screenshots (includes unrelated windows)
   - Keyboard shortcuts only (requires knowledge of shortcuts)
