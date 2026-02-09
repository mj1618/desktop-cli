# Feature: Implement `click` and `focus` Commands with macOS Input Simulation

## Priority: HIGH (Phase 2, Tasks 1-2 and 5 — first interactive commands)

## Problem

Agents can now list windows and read UI element trees, but they cannot interact with anything. The `click` and `focus` commands are the first input actions and the most critical for agent workflows. Without them, agents can see buttons but can't press them.

Both commands are currently stubs returning "not yet implemented". The `Inputter` and `WindowManager` interfaces exist but have no macOS backend implementations. The CGo infrastructure for input simulation (CGEvent-based mouse events) and window focusing (NSRunningApplication / AX API) needs to be built.

## What to Build

### 1. macOS Input Simulation — `internal/platform/darwin/inputter.go`

Implement the `platform.Inputter` interface using CGo + CoreGraphics event simulation.

```go
//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework ApplicationServices -framework Foundation
#include "input.h"
*/
import "C"

type DarwinInputter struct{}

func NewInputter() *DarwinInputter {
    return &DarwinInputter{}
}

func (inp *DarwinInputter) Click(x, y int, button platform.MouseButton, count int) error {
    // Map MouseButton to CGMouseButton enum
    // Create CGEventCreateMouseEvent for mouseDown + mouseUp
    // For double-click, set click count to 2
    // Post events to the system event stream
}

func (inp *DarwinInputter) MoveMouse(x, y int) error {
    // CGEventCreateMouseEvent with kCGEventMouseMoved
}

func (inp *DarwinInputter) Scroll(x, y int, dx, dy int) error {
    // Stub for now — return not implemented (will be done in scroll command task)
    return fmt.Errorf("scroll not yet implemented")
}

func (inp *DarwinInputter) Drag(fromX, fromY, toX, toY int) error {
    // Stub for now — return not implemented
    return fmt.Errorf("drag not yet implemented")
}

func (inp *DarwinInputter) TypeText(text string, delayMs int) error {
    // Stub for now — return not implemented (will be done in type command task)
    return fmt.Errorf("type not yet implemented")
}

func (inp *DarwinInputter) KeyCombo(keys []string) error {
    // Stub for now — return not implemented
    return fmt.Errorf("key combo not yet implemented")
}
```

### 2. C Input Helpers — `internal/platform/darwin/input.h` and `input.c`

```c
// input.h
#ifndef INPUT_H
#define INPUT_H

#include <CoreGraphics/CoreGraphics.h>

// Click at screen coordinates.
// button: 0=left, 1=right, 2=middle
// count: 1=single click, 2=double click
int cg_click(float x, float y, int button, int count);

// Move mouse to screen coordinates.
int cg_move_mouse(float x, float y);

#endif
```

```c
// input.c
#include "input.h"
#include <unistd.h>

int cg_click(float x, float y, int button, int count) {
    CGPoint point = CGPointMake(x, y);

    CGEventType downType, upType;
    CGMouseButton cgButton;

    switch (button) {
        case 1:  // right
            downType = kCGEventRightMouseDown;
            upType = kCGEventRightMouseUp;
            cgButton = kCGMouseButtonRight;
            break;
        case 2:  // middle
            downType = kCGEventOtherMouseDown;
            upType = kCGEventOtherMouseUp;
            cgButton = kCGMouseButtonCenter;
            break;
        default:  // left
            downType = kCGEventLeftMouseDown;
            upType = kCGEventLeftMouseUp;
            cgButton = kCGMouseButtonLeft;
            break;
    }

    CGEventRef downEvent = CGEventCreateMouseEvent(NULL, downType, point, cgButton);
    CGEventRef upEvent = CGEventCreateMouseEvent(NULL, upType, point, cgButton);

    if (!downEvent || !upEvent) {
        if (downEvent) CFRelease(downEvent);
        if (upEvent) CFRelease(upEvent);
        return -1;
    }

    // Set click count for double-click
    if (count > 1) {
        CGEventSetIntegerValueField(downEvent, kCGMouseEventClickState, count);
        CGEventSetIntegerValueField(upEvent, kCGMouseEventClickState, count);
    }

    CGEventPost(kCGHIDEventTap, downEvent);
    CGEventPost(kCGHIDEventTap, upEvent);

    CFRelease(downEvent);
    CFRelease(upEvent);

    return 0;
}

int cg_move_mouse(float x, float y) {
    CGPoint point = CGPointMake(x, y);
    CGEventRef event = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, point, kCGMouseButtonLeft);
    if (!event) return -1;
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
    return 0;
}
```

### 3. macOS Window Manager — `internal/platform/darwin/window_manager.go`

Implement the `platform.WindowManager` interface using NSRunningApplication and AX API.

```go
//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework ApplicationServices -framework Foundation
#include "window_focus.h"
*/
import "C"

type DarwinWindowManager struct {
    reader *DarwinReader  // reuse for window lookup
}

func NewWindowManager(reader *DarwinReader) *DarwinWindowManager {
    return &DarwinWindowManager{reader: reader}
}

func (wm *DarwinWindowManager) FocusWindow(opts platform.FocusOptions) error {
    // Resolve PID from opts (--app, --pid, --window, --window-id)
    // Use NSRunningApplication to activate the app (bring to front)
    // If --window or --window-id specified, also raise specific window via AX API
}

func (wm *DarwinWindowManager) GetFrontmostApp() (string, int, error) {
    // Use NSWorkspace.frontmostApplication
    // Return app name and PID
}
```

### 4. C Window Focus Helpers — `internal/platform/darwin/window_focus.h` and `window_focus.c`

```c
// window_focus.h
#ifndef WINDOW_FOCUS_H
#define WINDOW_FOCUS_H

#include <ApplicationServices/ApplicationServices.h>

// Activate (focus) an application by PID.
// Returns 0 on success, -1 on failure.
int ax_focus_app(pid_t pid);

// Raise a specific window by PID and window index (from AX API).
// If windowTitle is non-NULL, matches by title substring.
// If windowID > 0, matches by CGWindowID.
// Returns 0 on success, -1 on failure.
int ax_raise_window(pid_t pid, const char* windowTitle, int windowID);

// Get the frontmost application's name and PID.
// Caller must free appName.
int ax_get_frontmost_app(char** appName, pid_t* pid);

#endif
```

The C implementation should use:
- `NSRunningApplication *app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid]` and `[app activateWithOptions:NSApplicationActivateIgnoringOtherApps]` to bring an app to the foreground
- AX API to enumerate windows and call `AXUIElementPerformAction(window, kAXRaiseAction)` to raise a specific window
- `NSWorkspace.sharedWorkspace.frontmostApplication` for GetFrontmostApp

### 5. Register Backends in Provider — Update `internal/platform/darwin/init.go`

Currently, `init()` only registers a Reader. Add the Inputter and WindowManager:

```go
func init() {
    platform.NewProviderFunc = func() (*platform.Provider, error) {
        reader := NewReader()
        return &platform.Provider{
            Reader:        reader,
            Inputter:      NewInputter(),
            WindowManager: NewWindowManager(reader),
        }, nil
    }
}
```

### 6. Wire `click` Command — `cmd/click.go`

Replace the stub with real logic:

```go
func runClick(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }

    id, _ := cmd.Flags().GetInt("id")
    x, _ := cmd.Flags().GetInt("x")
    y, _ := cmd.Flags().GetInt("y")
    buttonStr, _ := cmd.Flags().GetString("button")
    double, _ := cmd.Flags().GetBool("double")
    appName, _ := cmd.Flags().GetString("app")
    window, _ := cmd.Flags().GetString("window")

    button, err := platform.ParseMouseButton(buttonStr)
    if err != nil {
        return err
    }

    count := 1
    if double {
        count = 2
    }

    if id != 0 {
        // Element ID click: re-read the tree, find element by ID, compute center
        if appName == "" && window == "" {
            return fmt.Errorf("--app or --window is required when using --id")
        }
        opts := platform.ReadOptions{
            App:    appName,
            Window: window,
        }
        elements, err := provider.Reader.ReadElements(opts)
        if err != nil {
            return err
        }
        elem := model.FindElementByID(elements, id)
        if elem == nil {
            return fmt.Errorf("element with ID %d not found", id)
        }
        // Compute center of element bounds
        cx := elem.Bounds[0] + elem.Bounds[2]/2
        cy := elem.Bounds[1] + elem.Bounds[3]/2
        return provider.Inputter.Click(cx, cy, button, count)
    }

    if x == 0 && y == 0 {
        return fmt.Errorf("specify either --id or --x/--y coordinates")
    }

    return provider.Inputter.Click(x, y, button, count)
}
```

### 7. Wire `focus` Command — `cmd/focus.go`

Replace the stub with real logic:

```go
func runFocus(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }

    appName, _ := cmd.Flags().GetString("app")
    window, _ := cmd.Flags().GetString("window")
    windowID, _ := cmd.Flags().GetInt("window-id")
    pid, _ := cmd.Flags().GetInt("pid")

    if appName == "" && window == "" && windowID == 0 && pid == 0 {
        return fmt.Errorf("specify at least one of --app, --window, --window-id, or --pid")
    }

    opts := platform.FocusOptions{
        App:      appName,
        Window:   window,
        WindowID: windowID,
        PID:      pid,
    }

    return provider.WindowManager.FocusWindow(opts)
}
```

### 8. Helper: `model.FindElementByID()` — Add to `internal/model/element.go`

A recursive function to find an element by its ID in a tree:

```go
// FindElementByID searches the element tree for an element with the given ID.
// Returns nil if not found.
func FindElementByID(elements []Element, id int) *Element {
    for i := range elements {
        if elements[i].ID == id {
            return &elements[i]
        }
        if found := FindElementByID(elements[i].Children, id); found != nil {
            return found
        }
    }
    return nil
}
```

### 9. Unit Tests

- `internal/model/element_test.go` — Add test for `FindElementByID` (found at root, found nested, not found)
- `cmd/click_test.go` — Test flag parsing, validate that --id requires --app or --window
- `cmd/focus_test.go` — Test flag parsing, validate that at least one target flag is required

## Files to Create

- `internal/platform/darwin/inputter.go` — DarwinInputter implementing Inputter interface
- `internal/platform/darwin/input.h` — C header for mouse event functions
- `internal/platform/darwin/input.c` — C implementation of mouse click and move
- `internal/platform/darwin/window_manager.go` — DarwinWindowManager implementing WindowManager interface
- `internal/platform/darwin/window_focus.h` — C header for window focus functions
- `internal/platform/darwin/window_focus.c` — C implementation of app activation and window raising

## Files to Modify

- `internal/platform/darwin/init.go` — Register Inputter and WindowManager in provider
- `cmd/click.go` — Replace stub with real implementation
- `cmd/focus.go` — Replace stub with real implementation
- `internal/model/element.go` — Add `FindElementByID()` helper
- `internal/model/element_test.go` — Add tests for FindElementByID

## Acceptance Criteria

- [ ] `go build ./...` succeeds on macOS
- [ ] `go build ./...` succeeds on non-macOS (stubs return errors for unimplemented methods)
- [ ] `go test ./...` passes
- [ ] `desktop-cli click --x 100 --y 200` clicks at absolute coordinates
- [ ] `desktop-cli click --x 100 --y 200 --button right` right-clicks
- [ ] `desktop-cli click --x 100 --y 200 --double` double-clicks
- [ ] `desktop-cli click --id 5 --app "Finder"` re-reads the element tree, finds element 5, clicks its center
- [ ] `desktop-cli click --id 999 --app "Finder"` returns a clear "element not found" error
- [ ] `desktop-cli click --id 5` (without --app) returns a clear error about needing --app or --window
- [ ] `desktop-cli click` (no flags) returns a clear error about needing --id or --x/--y
- [ ] `desktop-cli focus --app "Finder"` brings Finder to the foreground
- [ ] `desktop-cli focus --pid <pid>` brings the app with that PID to the foreground
- [ ] `desktop-cli focus --window "Downloads"` focuses the window matching that title
- [ ] `desktop-cli focus` (no flags) returns a clear error
- [ ] Provider now includes Inputter and WindowManager (not just Reader)
- [ ] `FindElementByID` works for root elements, nested elements, and returns nil for missing IDs
- [ ] No memory leaks in C code (all CGEvent objects released)
- [ ] README.md and SKILL.md updated if relevant

## Implementation Notes

- **CGEvent permissions**: Mouse event posting requires the same accessibility permission as reading. The `CheckAccessibilityPermission()` call should be made at the start of `Click()`.
- **Coordinate system**: macOS uses a coordinate system where (0,0) is the top-left of the primary display. `CGWindowListCopyWindowInfo` returns bounds in this coordinate system, and `CGEventCreateMouseEvent` expects coordinates in this same system. No conversion needed.
- **Double-click timing**: CGEvent handles double-click timing internally when you set `kCGMouseEventClickState` to 2. You need to post two rapid down+up pairs.
- **Focus before click**: When using `--id`, the command should focus the target app/window before clicking to ensure the click lands in the right place. Use the WindowManager to focus first.
- **Element ID stability**: IDs are assigned by traversal order during `read`. When `click --id` re-reads the tree, the same traversal produces the same IDs as long as the UI hasn't changed. This is the stateless approach from PLAN.md.
- **AppKit vs CGo for focus**: `NSRunningApplication.activateWithOptions:` is the modern API for bringing an app to front. This requires Objective-C via CGo. Alternatively, `AXUIElementPerformAction(app, kAXRaiseAction)` can raise windows. Use both: NSRunningApplication to activate the app, then AX API to raise a specific window if needed.
- **Framework linking**: The new C files need `-framework AppKit` in addition to the existing frameworks. Add it to the `#cgo LDFLAGS` in the relevant Go files.
