# Test Result

## Status
PASS

## Evidence

### All tests pass
```
$ go test ./...
ok  	github.com/mj1618/desktop-cli	(cached)
ok  	github.com/mj1618/desktop-cli/cmd	(cached)
ok  	github.com/mj1618/desktop-cli/internal/model	(cached)
ok  	github.com/mj1618/desktop-cli/internal/output	(cached)
ok  	github.com/mj1618/desktop-cli/internal/platform	(cached)
ok  	github.com/mj1618/desktop-cli/internal/platform/darwin	(cached)
```

### Build succeeds
```
$ go build -o desktop-cli .
(no errors)
```

### Baseline: screenshot WITHOUT --include-menubar
```
$ ./desktop-cli screenshot --app "TextEdit" --output /tmp/test-no-menubar.png
```
Visual result: Only shows the TextEdit window (title bar, toolbar, ruler, text area). No menu bar visible.

### Fix: screenshot WITH --include-menubar
```
$ ./desktop-cli screenshot --app "TextEdit" --include-menubar --output /tmp/test-with-menubar.png
```
Visual result: Shows the full macOS menu bar at the top (Apple logo, TextEdit, File, Edit, Format, View, Window, Help, plus system tray icons) AND the app window below it. Menu items are clearly visible and identifiable.

### screenshot-coords WITH --include-menubar
```
$ ./desktop-cli screenshot-coords --app "TextEdit" --include-menubar --output /tmp/test-coords-menubar.png
```
Visual result: Shows the menu bar with coordinate annotations on menu items (e.g. (731,227), (784,227)) plus the app window with element coordinates. This enables coordinate-based clicking on menu items.

### Confirmed original problem still exists without the flag
```
$ ./desktop-cli read --app "TextEdit" --roles "menu,menuitem" --format agent
# TextEdit
(empty - no menu items returned)
```
This confirms menu items are not exposed in the accessibility tree, making the `--include-menubar` screenshot flag the correct workaround.

### Flag is available on both screenshot commands
- `screenshot --include-menubar` works
- `screenshot-coords --include-menubar` works

## Notes
- The `--include-menubar` flag only takes effect when combined with `--app` (or `--pid`/`--window`), which is the correct behavior since full-screen screenshots already include the menu bar.
- The composite image correctly shows the menu bar at the top and the window content below, with coordinates remaining accurate for clicking purposes.
- One minor edge case: when TextEdit shows an Open dialog (instead of an editing window), `screenshot-coords --include-menubar` still works correctly, capturing the dialog with the menu bar above it.
- No regressions observed in existing screenshot behavior when `--include-menubar` is not specified.
