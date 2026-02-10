# Feature: Implement `read` Command with macOS Accessibility Element Tree Traversal

## Priority: CRITICAL (Phase 1, Task 3 — the core feature of the entire tool)

## Problem

The `read` command is the most important command in desktop-cli. It's what allows agents to *see* UI elements — buttons, text fields, links, labels — in any application window. Without it, agents can list windows but cannot inspect or understand any UI content.

The `read` command is currently a stub returning "not yet implemented". The `DarwinReader.ReadElements()` method is also a stub. The entire CGo + Accessibility API bridge for element tree traversal needs to be built.

## What to Build

### 1. C Accessibility Bridge — `internal/platform/darwin/accessibility.h` and `accessibility.c`

A C implementation that traverses the macOS accessibility tree using the AX API and returns a flat array of element info structs. The Go code will then reassemble these into the tree structure.

#### `accessibility.h`

```c
#ifndef ACCESSIBILITY_H
#define ACCESSIBILITY_H

#include <ApplicationServices/ApplicationServices.h>

typedef struct {
    int id;
    char* role;
    char* title;
    char* value;
    char* description;
    float x, y, width, height;
    int enabled;
    int focused;
    int selected;
    int parentID;    // -1 for root elements
    int actionCount;
    char** actions;
} AXElementInfo;

// Read the element tree for a given app PID.
// If windowTitle is non-NULL, filters to windows matching that substring.
// If windowID > 0, filters to the specific window ID.
// maxDepth of 0 means unlimited.
// Returns 0 on success, -1 on failure.
int ax_read_elements(pid_t pid, const char* windowTitle, int windowID,
                     int maxDepth, AXElementInfo** outElements, int* outCount);

// Free the element array returned by ax_read_elements.
void ax_free_elements(AXElementInfo* elements, int count);

#endif
```

#### `accessibility.c` — Key Implementation Details

The C code should:

1. **Create AX app reference**: `AXUIElementCreateApplication(pid)`
2. **Get windows**: `AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, &windows)`
3. **Filter windows** by title substring or window ID if specified
4. **Traverse each window's element tree recursively**:
   - For each element, read these attributes:
     - `kAXRoleAttribute` → role string
     - `kAXTitleAttribute` → title/label
     - `kAXValueAttribute` → current value (text content, checkbox state, etc.)
     - `kAXDescriptionAttribute` → accessibility description
     - `kAXPositionAttribute` → screen position (via `AXValueGetValue` with `kAXValueCGPointType`)
     - `kAXSizeAttribute` → element size (via `AXValueGetValue` with `kAXValueCGSizeType`)
     - `kAXEnabledAttribute` → whether interactive
     - `kAXFocusedAttribute` → has keyboard focus
     - `kAXSelectedAttribute` → is selected
     - `kAXChildrenAttribute` → child elements (recurse)
     - `kAXActionsAttribute` → available actions (via `AXUIElementCopyActionNames`)
   - Assign sequential integer IDs (starting from 1) during breadth-first or depth-first traversal
   - Record `parentID` for each element (-1 for root/window-level elements)
   - Respect `maxDepth` — stop recursing beyond the limit
5. **Return a flat C array** of `AXElementInfo` structs. Go code rebuilds the tree using `parentID`.

Important CGo/C notes:
- Use `CFRelease` for all CF types after use to avoid memory leaks
- Handle NULL/nil returns gracefully — some elements may not have all attributes
- Strings from CF should be converted with a helper like `cfstring_to_cstring()` (already exists in `window_list.c`, can share or duplicate)
- Actions should be mapped from AX names (e.g., "AXPress") to short names (e.g., "press")

#### Action Name Mapping (in C)

```
AXPress → press
AXCancel → cancel
AXPick → pick
AXIncrement → increment
AXDecrement → decrement
AXConfirm → confirm
AXShowMenu → showmenu
```

### 2. Go Element Reconstruction — Extend `internal/platform/darwin/reader.go`

Implement `DarwinReader.ReadElements()`:

```go
func (r *DarwinReader) ReadElements(opts platform.ReadOptions) ([]model.Element, error) {
    if err := CheckAccessibilityPermission(); err != nil {
        return nil, err
    }

    // Determine target PID
    pid, windowTitle, windowID := resolvePIDAndWindow(opts)
    if pid == 0 {
        return nil, fmt.Errorf("no target specified: use --app, --pid, --window, or --window-id")
    }

    // Call C accessibility bridge
    var cElements *C.AXElementInfo
    var cCount C.int
    cWindowTitle := (*C.char)(nil)
    if windowTitle != "" {
        cWindowTitle = C.CString(windowTitle)
        defer C.free(unsafe.Pointer(cWindowTitle))
    }

    if C.ax_read_elements(C.pid_t(pid), cWindowTitle, C.int(windowID),
        C.int(opts.Depth), &cElements, &cCount) != 0 {
        return nil, fmt.Errorf("failed to read accessibility tree for PID %d", pid)
    }
    defer C.ax_free_elements(cElements, cCount)

    // Convert flat C array to Go element tree
    elements := buildElementTree(cElements, cCount)

    // Apply role and bbox filters
    var bbox *[4]int
    if opts.BBox != nil {
        b := [4]int{opts.BBox.X, opts.BBox.Y, opts.BBox.Width, opts.BBox.Height}
        bbox = &b
    }
    elements = model.FilterElements(elements, opts.Roles, bbox)

    return elements, nil
}
```

#### Helper: `resolvePIDAndWindow()`

This helper resolves the target PID from `--app`, `--pid`, `--window`, or `--window-id` flags:

```go
func (r *DarwinReader) resolvePIDAndWindow(opts platform.ReadOptions) (pid int, windowTitle string, windowID int) {
    if opts.PID != 0 {
        return opts.PID, opts.Window, opts.WindowID
    }
    if opts.App != "" {
        // Use ListWindows to find PID for the app
        windows, err := r.ListWindows(platform.ListOptions{App: opts.App})
        if err != nil || len(windows) == 0 {
            return 0, "", 0
        }
        // If --window is also specified, find the matching window
        for _, w := range windows {
            if opts.Window != "" && strings.Contains(strings.ToLower(w.Title), strings.ToLower(opts.Window)) {
                return w.PID, "", w.ID  // Use window ID for precision
            }
        }
        // Return first window's PID
        return windows[0].PID, opts.Window, opts.WindowID
    }
    if opts.WindowID != 0 {
        // Need to find PID from window ID via ListWindows
        windows, err := r.ListWindows(platform.ListOptions{})
        if err != nil {
            return 0, "", 0
        }
        for _, w := range windows {
            if w.ID == opts.WindowID {
                return w.PID, "", w.ID
            }
        }
    }
    return 0, "", 0
}
```

#### Helper: `buildElementTree()`

Converts the flat C array (with `parentID` references) into a nested Go `[]model.Element` tree:

```go
func buildElementTree(cElements *C.AXElementInfo, cCount C.int) []model.Element {
    count := int(cCount)
    cSlice := unsafe.Slice(cElements, count)

    // Build flat list of elements
    elemMap := make(map[int]*model.Element, count)
    var roots []int

    for i := 0; i < count; i++ {
        ce := cSlice[i]
        id := int(ce.id)
        parentID := int(ce.parentID)

        // Map AX role to compact code
        role := model.MapRole(C.GoString(ce.role))

        // Build actions list
        var actions []string
        if ce.actionCount > 0 {
            cActions := unsafe.Slice(ce.actions, int(ce.actionCount))
            for j := 0; j < int(ce.actionCount); j++ {
                actions = append(actions, C.GoString(cActions[j]))
            }
        }

        // Build enabled pointer (nil = enabled, false = disabled)
        var enabled *bool
        if ce.enabled == 0 {
            f := false
            enabled = &f
        }

        elem := &model.Element{
            ID:          id,
            Role:        role,
            Title:       C.GoString(ce.title),
            Value:       C.GoString(ce.value),
            Description: C.GoString(ce.description),
            Bounds:      [4]int{int(ce.x), int(ce.y), int(ce.width), int(ce.height)},
            Focused:     ce.focused != 0,
            Enabled:     enabled,
            Selected:    ce.selected != 0,
            Actions:     actions,
        }
        elemMap[id] = elem

        if parentID < 0 {
            roots = append(roots, id)
        }
    }

    // Build tree by assigning children
    for i := 0; i < count; i++ {
        ce := cSlice[i]
        parentID := int(ce.parentID)
        id := int(ce.id)
        if parentID >= 0 {
            if parent, ok := elemMap[parentID]; ok {
                parent.Children = append(parent.Children, *elemMap[id])
            }
        }
    }

    // Collect root elements
    var result []model.Element
    for _, id := range roots {
        result = append(result, *elemMap[id])
    }
    return result
}
```

### 3. Wire `read` Command — `cmd/read.go`

Replace the `notImplemented("read")` stub with real logic:

```go
func runRead(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }

    appName, _ := cmd.Flags().GetString("app")
    window, _ := cmd.Flags().GetString("window")
    windowID, _ := cmd.Flags().GetInt("window-id")
    pid, _ := cmd.Flags().GetInt("pid")
    depth, _ := cmd.Flags().GetInt("depth")
    rolesStr, _ := cmd.Flags().GetString("roles")
    visibleOnly, _ := cmd.Flags().GetBool("visible-only")
    bboxStr, _ := cmd.Flags().GetString("bbox")
    compact, _ := cmd.Flags().GetBool("compact")
    pretty, _ := cmd.Flags().GetBool("pretty")

    // Parse roles
    var roles []string
    if rolesStr != "" {
        for _, r := range strings.Split(rolesStr, ",") {
            roles = append(roles, strings.TrimSpace(r))
        }
    }

    // Parse bbox
    var bbox *platform.Bounds
    if bboxStr != "" {
        bbox, err = platform.ParseBBox(bboxStr)
        if err != nil {
            return err
        }
    }

    opts := platform.ReadOptions{
        App:         appName,
        Window:      window,
        WindowID:    windowID,
        PID:         pid,
        Depth:       depth,
        Roles:       roles,
        VisibleOnly: visibleOnly,
        BBox:        bbox,
        Compact:     compact,
    }

    elements, err := provider.Reader.ReadElements(opts)
    if err != nil {
        return err
    }

    // Build ReadResult with metadata
    result := output.ReadResult{
        App:      appName,
        PID:      pid,
        TS:       time.Now().Unix(),
        Elements: elements,
    }

    return output.PrintJSON(result, pretty)
}
```

### 4. CGo Linker Flags

The `accessibility.c` file needs these frameworks linked:
- `ApplicationServices` (for AX API)
- `CoreFoundation` (for CF types)
- `CoreGraphics` (for CGPoint/CGSize from AXValue)

These are already specified in `reader.go`'s CGo preamble. The new `.c`/`.h` files just need to be in the same package directory and they'll be compiled automatically by CGo.

## Files to Create

- `internal/platform/darwin/accessibility.h` — C header for AX element traversal structs and functions
- `internal/platform/darwin/accessibility.c` — C implementation of `ax_read_elements()` and `ax_free_elements()`

## Files to Modify

- `internal/platform/darwin/reader.go` — Implement `ReadElements()`, add `resolvePIDAndWindow()` and `buildElementTree()` helpers, add `#include "accessibility.h"` to CGo preamble
- `cmd/read.go` — Replace stub with real `runRead()` implementation wired to platform provider

## Acceptance Criteria

- [ ] `go build ./...` succeeds on macOS
- [ ] `go build ./...` succeeds on non-macOS (ReadElements returns "not implemented" or "unsupported platform")
- [ ] `go test ./...` passes
- [ ] `desktop-cli read --app "Finder"` outputs a JSON element tree for Finder's frontmost window
- [ ] `desktop-cli read --app "Finder" --depth 2` limits traversal depth
- [ ] `desktop-cli read --app "Finder" --roles "btn,txt"` only includes buttons and text elements
- [ ] `desktop-cli read --app "Finder" --pretty` pretty-prints the JSON output
- [ ] `desktop-cli read --pid <pid>` reads elements for a process by PID
- [ ] `desktop-cli read --window "Downloads"` reads elements for a window matching the title
- [ ] Element IDs are sequential integers starting from 1
- [ ] Roles are mapped to compact codes (btn, txt, lnk, etc.) per `model.RoleMap`
- [ ] Empty/default fields are omitted from JSON (title, value, description when empty; focused/selected when false; enabled when true/nil)
- [ ] Bounds are `[x, y, width, height]` integer arrays
- [ ] Actions are short names (press, cancel, pick, etc.) not AX names
- [ ] Without --app, --pid, --window, or --window-id, returns a helpful error
- [ ] Accessibility permission denied returns clear instructions
- [ ] No memory leaks in C code (all CF objects released, all malloc'd strings freed)
- [ ] README.md and SKILL.md updated if needed

## Implementation Notes

- The flat-array-with-parentID pattern (used by `ax_read_elements`) minimizes CGo crossings. One C call traverses the entire tree and returns everything. Go does the tree assembly. This is the same pattern described in PLAN.md.
- `maxDepth` filtering MUST happen in C during traversal, not in Go after the fact. This prevents reading enormous trees (e.g., browser web content with thousands of DOM nodes) when the agent only needs a shallow view.
- The `--visible-only` flag should use bounds checking: elements with zero-size bounds or bounds entirely off-screen should be excluded during C traversal.
- Actions like `AXPress` should be mapped to short names in Go after retrieval, keeping the C code simpler.
- Error handling: If the AX API returns errors for individual elements (common for protected/system elements), skip those elements silently rather than failing the entire read.
- The CGo preamble in `reader.go` already links the required frameworks. The new `.c` and `.h` files in the same `darwin` package directory will be picked up automatically.
