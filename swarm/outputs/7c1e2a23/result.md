# Test Result

## Status
FAIL

## Evidence

### Tests and Build
All unit tests pass and the binary builds successfully:
```
$ go test ./...
ok  	github.com/mj1618/desktop-cli	(cached)
ok  	github.com/mj1618/desktop-cli/cmd	(cached)
ok  	github.com/mj1618/desktop-cli/internal/model	(cached)
ok  	github.com/mj1618/desktop-cli/internal/output	(cached)
ok  	github.com/mj1618/desktop-cli/internal/platform	(cached)
```

### Reproduction — Issue Still Present
After opening System Settings and navigating to Date & Time:

```
$ ./desktop-cli read --app "System Settings" --roles "chk"
app: System Settings
ts: 1770714972
elements: []

$ ./desktop-cli read --app "System Settings" --roles "radio"
app: System Settings
ts: 1770714977
elements: []

$ ./desktop-cli read --app "System Settings" --roles "toggle"
app: System Settings
ts: 1770714977
elements: []
```

After clicking "Date & Time" (id 160), `read --format agent` still only shows sidebar buttons — no content pane controls (toggles, checkboxes, switches) appear.

The full unfiltered tree (186 elements) contains roles: row, cell, txt, btn, group, other, scroll, input, toolbar, img — but **no chk, toggle, radio, or switch roles**.

### Root Cause Analysis
This is an **OS-level accessibility limitation**, not a desktop-cli code issue:

1. **Code supports toggles**: `AXSwitch` is explicitly mapped to "toggle" in `internal/model/roles.go`
2. **Full-depth traversal**: The C code in `internal/platform/darwin/accessibility.c` does unlimited-depth recursive traversal with no role exclusions
3. **No filtering during collection**: All elements from the OS accessibility API are captured; role filtering happens post-traversal
4. **macOS doesn't expose these controls**: System Settings (a SwiftUI app) simply doesn't expose its toggle switches through the standard accessibility API as `AXSwitch`, `AXCheckBox`, or `AXRadioButton` roles

## Notes
- This issue cannot be fixed in desktop-cli code — it requires Apple to improve accessibility support in System Settings
- The improvement.md proposes changes ("ensure toggle switches are properly discovered") that are outside the tool's control since it faithfully reads whatever the OS accessibility API exposes
- A potential workaround (not a fix) could be screenshot-based detection of toggle positions, but that would be a significant architectural addition
- This bug should be categorized as a **platform limitation / won't fix** rather than a code bug
