# Feature: Implement `action` Command for Direct Accessibility Actions

## Priority: HIGH (Phase 4 — direct AX action execution for reliable element interaction)

## Problem

Agents using `desktop-cli` can see available accessibility actions on elements (via the `a` field in `read` output — e.g. `press`, `cancel`, `pick`, `increment`, `decrement`), but there's no command to **execute** those actions directly. Currently, to "press" a button, agents must:

1. `read` to find the element and its bounds
2. `click --id <id>` which re-reads the tree, computes center coordinates, and simulates a mouse click

This coordinate-based approach has significant limitations:

1. **Unreliable for occluded elements** — If another window or popup covers the target element, the mouse click hits the wrong thing
2. **Fragile for off-screen elements** — Elements in scroll containers that aren't currently visible can't be clicked by coordinates
3. **Slower** — Requires mouse movement, which can trigger hover states, tooltips, or other side effects
4. **Limited action types** — Mouse clicks can only do `press`. They can't `cancel`, `pick` (for dropdown selection), `increment`/`decrement` (for steppers/sliders), `confirm`, `showMenu`, etc.
5. **Race conditions** — Between reading element bounds and clicking, the UI might shift (animation, scroll, resize)

The macOS Accessibility API provides `AXUIElementPerformAction()` which executes actions directly on the element reference, bypassing coordinates entirely. This is the same mechanism that VoiceOver and other assistive technologies use — it's the most reliable way to interact with UI elements.

## What to Build

### 1. Add ActionPerformer Interface — `internal/platform/platform.go`

Add a new interface for performing accessibility actions on elements:

```go
// ActionPerformer performs accessibility actions directly on UI elements.
type ActionPerformer interface {
    // PerformAction executes an accessibility action on an element identified
    // by its sequential ID within the given read scope.
    // The action string matches the `a` field values from read output (e.g. "press", "cancel", "pick").
    PerformAction(opts ActionOptions) error
}

// ActionOptions configures which element to act on and what action to perform.
type ActionOptions struct {
    App      string // Scope to application
    Window   string // Scope to window
    WindowID int    // Scope to window by system ID
    PID      int    // Scope to process
    ID       int    // Element ID (from read output)
    Action   string // Action to perform: "press", "cancel", "pick", "increment", "decrement", "confirm", "showMenu", "raise"
    Value    string // Optional value for actions that take a parameter (e.g. "pick" with a value to select)
}
```

Add `ActionPerformer` to the `Provider` struct in `internal/platform/provider.go`:

```go
type Provider struct {
    Reader          Reader
    Inputter        Inputter
    WindowManager   WindowManager
    Screenshotter   Screenshotter
    ActionPerformer ActionPerformer
}
```

### 2. C Implementation — `internal/platform/darwin/action.c` + `action.h`

**`action.h`:**
```c
#ifndef ACTION_H
#define ACTION_H

#include <ApplicationServices/ApplicationServices.h>

// Perform an accessibility action on the element at the given traversal index
// within the specified app/window.
// Returns 0 on success, -1 on failure.
int ax_perform_action(pid_t pid, int windowIndex, int elementIndex,
                      const char* actionName);

#endif
```

**`action.c` implementation approach:**

1. Get the app's AX reference: `AXUIElementCreateApplication(pid)`
2. Get the window at the given index (same traversal as `ax_read_elements`)
3. Traverse the element tree in the same order as `ax_read_elements` to find the element at the given sequential index (ID)
4. Once the element is found, call `AXUIElementPerformAction(element, actionCFString)`
5. Return success/failure

The key insight is that `AXUIElementPerformAction` works on the `AXUIElementRef` directly — no coordinates needed. The traversal order MUST match the order used in `ax_read_elements` so that element IDs are consistent between `read` and `action`.

**Action name mapping** — The `a` field in read output uses short names. Map these back to AX action constants:
- `press` → `kAXPressAction` (AXPress)
- `cancel` → `kAXCancelAction` (AXCancel)
- `pick` → `kAXPickAction` (AXPick) — for menus, dropdowns
- `increment` → `kAXIncrementAction` (AXIncrement) — for sliders, steppers
- `decrement` → `kAXDecrementAction` (AXDecrement)
- `confirm` → `kAXConfirmAction` (AXConfirm)
- `showMenu` → `kAXShowMenuAction` (AXShowMenu) — right-click context menu
- `raise` → `kAXRaiseAction` (AXRaise) — bring to front

If the short name doesn't match a known mapping, pass it through directly as a CFString (the AX API accepts string action names).

### 3. Go Wrapper — `internal/platform/darwin/action_performer.go`

```go
//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework Foundation
#include "action.h"
#include <stdlib.h>
*/
import "C"

import (
    "fmt"
    "unsafe"

    "github.com/mj1618/desktop-cli/internal/platform"
)

type DarwinActionPerformer struct {
    reader *DarwinReader
}

func NewActionPerformer(reader *DarwinReader) *DarwinActionPerformer {
    return &DarwinActionPerformer{reader: reader}
}

func (p *DarwinActionPerformer) PerformAction(opts platform.ActionOptions) error {
    if opts.ID <= 0 {
        return fmt.Errorf("--id is required")
    }
    if opts.Action == "" {
        return fmt.Errorf("--action is required")
    }

    // Resolve the target window using the reader (same pattern as click command)
    windows, err := p.reader.ListWindows(platform.ListOptions{
        App:      opts.App,
        Window:   opts.Window,
        WindowID: opts.WindowID,
        PID:      opts.PID,
    })
    if err != nil {
        return fmt.Errorf("failed to list windows: %w", err)
    }
    if len(windows) == 0 {
        return fmt.Errorf("no matching window found")
    }

    win := windows[0]

    // Map short action name to full AX action name
    axAction := mapActionName(opts.Action)

    cAction := C.CString(axAction)
    defer C.free(unsafe.Pointer(cAction))

    // windowIndex: we need to determine which window index this is within the app.
    // The reader's ListWindows returns the window with its system ID.
    // We'll need a helper in the C layer to find the window by its system ID within the app.
    // For now, use windowIndex 0 (first window of the app), matching the same
    // resolution logic as click/type/scroll commands.

    rc := C.ax_perform_action(C.int(win.PID), C.int(0), C.int(opts.ID), cAction)
    if rc != 0 {
        return fmt.Errorf("failed to perform action %q on element %d", opts.Action, opts.ID)
    }

    return nil
}

func mapActionName(short string) string {
    switch short {
    case "press":
        return "AXPress"
    case "cancel":
        return "AXCancel"
    case "pick":
        return "AXPick"
    case "increment":
        return "AXIncrement"
    case "decrement":
        return "AXDecrement"
    case "confirm":
        return "AXConfirm"
    case "showMenu":
        return "AXShowMenu"
    case "raise":
        return "AXRaise"
    default:
        return short // Pass through if not a known short name
    }
}
```

### 4. Command Definition — `cmd/action.go` (new file)

```go
package cmd

import (
    "fmt"

    "github.com/mj1618/desktop-cli/internal/output"
    "github.com/mj1618/desktop-cli/internal/platform"
    "github.com/spf13/cobra"
)

type ActionResult struct {
    OK     bool   `yaml:"ok"`
    Action string `yaml:"action"`
    ID     int    `yaml:"id"`
    Name   string `yaml:"name"`
}

var actionCmd = &cobra.Command{
    Use:   "action",
    Short: "Perform an accessibility action on a UI element",
    Long: `Execute an accessibility action directly on a UI element by ID.

Actions are the same as shown in the 'a' field of 'read' output:
  press      - Press/activate the element (buttons, links, menu items)
  cancel     - Cancel the current operation
  pick       - Pick/select (dropdowns, menus)
  increment  - Increase value (sliders, steppers)
  decrement  - Decrease value (sliders, steppers)
  confirm    - Confirm a dialog or selection
  showMenu   - Show context menu for the element
  raise      - Bring element/window to front

Unlike 'click', this does NOT simulate mouse events — it calls the accessibility
action directly on the element, which works even for off-screen or occluded elements.`,
    RunE: runAction,
}

func init() {
    rootCmd.AddCommand(actionCmd)
    actionCmd.Flags().Int("id", 0, "Element ID from read output (required)")
    actionCmd.Flags().String("action", "press", "Action to perform (default: press)")
    actionCmd.Flags().String("app", "", "Scope to application")
    actionCmd.Flags().String("window", "", "Scope to window")
    actionCmd.Flags().Int("window-id", 0, "Scope to window by system ID")
    actionCmd.Flags().Int("pid", 0, "Scope to process by PID")
    actionCmd.MarkFlagRequired("id")
}

func runAction(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }
    if provider.ActionPerformer == nil {
        return fmt.Errorf("action not supported on this platform")
    }

    id, _ := cmd.Flags().GetInt("id")
    action, _ := cmd.Flags().GetString("action")
    appName, _ := cmd.Flags().GetString("app")
    window, _ := cmd.Flags().GetString("window")
    windowID, _ := cmd.Flags().GetInt("window-id")
    pid, _ := cmd.Flags().GetInt("pid")

    opts := platform.ActionOptions{
        App:      appName,
        Window:   window,
        WindowID: windowID,
        PID:      pid,
        ID:       id,
        Action:   action,
    }

    err = provider.ActionPerformer.PerformAction(opts)
    if err != nil {
        return err
    }

    return output.PrintYAML(ActionResult{
        OK:     true,
        Action: "action",
        ID:     id,
        Name:   action,
    })
}
```

### 5. Update C Code — `internal/platform/darwin/action.c`

The C implementation must traverse the element tree in the **exact same order** as `ax_read_elements` in `accessibility.c`. This is critical for element ID consistency.

Implementation approach:

```c
#include "action.h"
#include <stdlib.h>

// Recursive traversal counter — finds element at given index
static AXUIElementRef find_element_at_index(AXUIElementRef root, int targetIndex, int* currentIndex) {
    (*currentIndex)++;
    if (*currentIndex == targetIndex) {
        CFRetain(root);
        return root;
    }

    CFArrayRef children = NULL;
    AXUIElementCopyAttributeValue(root, kAXChildrenAttribute, (CFTypeRef*)&children);
    if (children == NULL) return NULL;

    CFIndex count = CFArrayGetCount(children);
    AXUIElementRef found = NULL;
    for (CFIndex i = 0; i < count && found == NULL; i++) {
        AXUIElementRef child = (AXUIElementRef)CFArrayGetValueAtIndex(children, i);
        found = find_element_at_index(child, targetIndex, currentIndex);
    }

    CFRelease(children);
    return found;
}

int ax_perform_action(pid_t pid, int windowIndex, int elementIndex, const char* actionName) {
    AXUIElementRef app = AXUIElementCreateApplication(pid);
    if (app == NULL) return -1;

    // Get the window
    CFArrayRef windows = NULL;
    AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, (CFTypeRef*)&windows);
    if (windows == NULL || CFArrayGetCount(windows) <= windowIndex) {
        if (windows) CFRelease(windows);
        CFRelease(app);
        return -1;
    }

    AXUIElementRef window = (AXUIElementRef)CFArrayGetValueAtIndex(windows, windowIndex);

    // Find the element at the given traversal index
    int currentIndex = 0;
    AXUIElementRef element = find_element_at_index(window, elementIndex, &currentIndex);

    CFRelease(windows);
    CFRelease(app);

    if (element == NULL) return -1;

    // Perform the action
    CFStringRef action = CFStringCreateWithCString(kCFAllocatorDefault, actionName, kCFStringEncodingUTF8);
    AXError result = AXUIElementPerformAction(element, action);

    CFRelease(action);
    CFRelease(element);

    return (result == kAXErrorSuccess) ? 0 : -1;
}
```

**Critical: Traversal order consistency.** The `find_element_at_index` function MUST traverse children in the same order as `ax_read_elements` in `accessibility.c`. Review `accessibility.c` before implementing to ensure the traversal matches. If `accessibility.c` skips certain elements (e.g., invisible ones, or applies depth limits), the same logic must apply here. Since `action` always targets a specific element by ID from a previous `read`, the read options (depth, roles, visible-only) that were used during `read` may need to be replayed during `action` to ensure the same traversal order produces the same IDs.

**Simplification option**: Rather than replicating the full traversal with filters, an alternative approach is:
1. Re-read the full element tree (using the same ReadElements path)
2. Match by ID to get the element's bounds/attributes
3. Use those attributes to find the AXUIElementRef in a separate traversal
4. Perform the action

This is slightly slower but much safer against ID mismatch bugs.

### 6. Register ActionPerformer — `internal/platform/darwin/init.go`

```go
func init() {
    platform.NewProviderFunc = func() (*platform.Provider, error) {
        reader := NewReader()
        inputter := NewInputter()
        windowManager := NewWindowManager(reader)
        screenshotter := NewScreenshotter(reader)
        actionPerformer := NewActionPerformer(reader)
        return &platform.Provider{
            Reader:          reader,
            Inputter:        inputter,
            WindowManager:   windowManager,
            Screenshotter:   screenshotter,
            ActionPerformer: actionPerformer,
        }, nil
    }
}
```

### 7. Update Documentation

**README.md** — Add an "Perform accessibility actions" section:

```markdown
### Perform accessibility actions

```bash
# Press a button directly via accessibility API (more reliable than click)
desktop-cli action --id 5 --app "Safari"

# Press is the default action, but you can specify others:
desktop-cli action --id 5 --action press --app "Safari"

# Increment a slider/stepper
desktop-cli action --id 12 --action increment --app "System Settings"

# Decrement a value
desktop-cli action --id 12 --action decrement --app "System Settings"

# Show context menu for an element
desktop-cli action --id 8 --action showMenu --app "Finder"

# Pick/select a dropdown item
desktop-cli action --id 15 --action pick --app "Safari"

# Cancel a dialog or operation
desktop-cli action --id 3 --action cancel --app "Safari"
```
```

**SKILL.md** — Add action command to quick reference:

```markdown
### Perform accessibility actions

```bash
desktop-cli action --id 5 --app "Safari"
desktop-cli action --id 5 --action press --app "Safari"
desktop-cli action --id 12 --action increment --app "System Settings"
desktop-cli action --id 12 --action decrement --app "System Settings"
desktop-cli action --id 8 --action showMenu --app "Finder"
```
```

Update the "Agent Workflow" section to mention `action` as an alternative to `click`:

```markdown
## Agent Workflow

1. `list --windows` to find the target window
2. `read --app <name> --depth 3 --roles "btn,lnk,input,txt"` to get the element tree as YAML
3. Use the element `i` (id) field with:
   - `action --id <id> --app <name>` to press buttons, toggle checkboxes, etc. (preferred — works on occluded elements)
   - `click --id <id> --app <name>` to click at coordinates (fallback when action isn't available)
   - `type --id <id> --app <name> --text "..."` to type into fields
4. `wait --app <name> --for-text "..." --timeout 10` to wait for UI to update
5. Repeat read/act/wait loop as needed
```

## Files to Create

- `internal/platform/darwin/action.c` — C implementation of AXUIElementPerformAction traversal
- `internal/platform/darwin/action.h` — C header for action functions
- `internal/platform/darwin/action_performer.go` — Go `DarwinActionPerformer` implementing `platform.ActionPerformer`
- `cmd/action.go` — New `action` command

## Files to Modify

- `internal/platform/platform.go` — Add `ActionPerformer` interface and `ActionOptions` struct
- `internal/platform/types.go` — Add `ActionOptions` if types are defined here instead
- `internal/platform/provider.go` — Add `ActionPerformer` field to `Provider` struct
- `internal/platform/darwin/init.go` — Register `ActionPerformer` in provider
- `README.md` — Add action command usage section and update agent workflow
- `SKILL.md` — Add action command to quick reference and update agent workflow

## Acceptance Criteria

- [ ] `go build ./...` succeeds on macOS
- [ ] `go build ./...` succeeds on non-macOS (ActionPerformer is nil in provider, command returns helpful error)
- [ ] `go test ./...` passes
- [ ] `desktop-cli action --help` shows all flags with descriptions
- [ ] `desktop-cli action --id 5 --app "Finder"` performs the default "press" action on element 5
- [ ] `desktop-cli action --id 5 --action press --app "Finder"` explicitly performs "press"
- [ ] `desktop-cli action --id 12 --action increment --app "System Settings"` increments a slider/stepper
- [ ] `desktop-cli action --id 12 --action decrement --app "System Settings"` decrements a slider/stepper
- [ ] `desktop-cli action --id 8 --action showMenu --app "Finder"` shows context menu
- [ ] `desktop-cli action` with no `--id` returns a clear error message
- [ ] Invalid element ID returns error: "failed to perform action ... on element ..."
- [ ] Invalid action name returns error from AX API
- [ ] On success: outputs YAML with `ok: true`, action name, and element ID
- [ ] Element IDs are consistent between `read` and `action` (same traversal order)
- [ ] Works on elements that are off-screen (in scroll containers)
- [ ] Works on elements that are behind other windows (occluded)
- [ ] README.md and SKILL.md are updated with action command examples

## Implementation Notes

- **`AXUIElementPerformAction`** is the core macOS API. It takes an `AXUIElementRef` and a `CFStringRef` action name. Action names are constants like `kAXPressAction` ("AXPress"), `kAXCancelAction` ("AXCancel"), etc. The API works regardless of element visibility or position — it's the same mechanism VoiceOver uses.
- **Element ID consistency is critical**: The `action` command must traverse the element tree in the exact same order as `read` so that element IDs match. The safest approach is to reuse the same C traversal function (`ax_read_elements` from `accessibility.c`) and then perform the action on the found element. Review `accessibility.c` carefully to understand the traversal order, including any filtering or skipping logic.
- **Default action is "press"**: Most elements have "press" as their primary action. Making it the default means agents can just do `action --id 5 --app "Safari"` without specifying `--action` in the common case.
- **Why this is better than `click --id`**: The `click` command re-reads the tree to get coordinates and simulates a mouse event. This fails if the element is occluded, off-screen, or has moved. `action` works directly on the AX element reference — no coordinates, no mouse simulation, no race conditions.
- **When to still use `click`**: Some elements don't expose AX actions (the `a` field is empty in `read` output). Custom-drawn UI or web content sometimes lacks proper AX action support. In those cases, coordinate-based `click` is the fallback. Agents should check the `a` field first and prefer `action` when available.
- **Permission**: `AXUIElementPerformAction` uses the same Accessibility permission as `read` and `click`. No additional permission is needed.
- **Memory management**: `AXUIElementRef` objects are `CFTypeRef` types — must be properly retained/released. The traversal function should `CFRetain` the found element and the caller must `CFRelease` it after performing the action.
- **Error handling**: `AXUIElementPerformAction` returns `AXError`. Common errors:
  - `kAXErrorActionUnsupported` — element doesn't support this action
  - `kAXErrorInvalidUIElement` — element reference is stale (UI changed)
  - `kAXErrorCannotComplete` — app is busy or unresponsive
  Map these to clear error messages for agents.
