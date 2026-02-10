# Feature: Implement `click` Command with macOS CGEvent Input Simulation

## Priority: HIGH (Phase 2, Task 1 — first input action command)

## Problem

Agents can now list windows (`list`) and read UI elements (`read`), but they cannot *interact* with anything. The `click` command is the most fundamental input action — it lets agents press buttons, open links, activate text fields, and trigger UI actions. Without it, agents are read-only.

The `click` command is currently a stub returning "not yet implemented". The `Inputter` interface exists but has no macOS implementation. The entire CGEvent-based input simulation bridge needs to be built.

## Dependencies

- The `read` command must be working (currently in progress by another task). The `click --id` mode depends on `ReadElements()` to resolve element IDs to screen coordinates.
- If the `read` command is not yet complete when this task starts, implement `click --x --y` (coordinate mode) first, then `click --id` mode.

## What to Build

### 1. macOS Input Simulator — `internal/platform/darwin/inputter.go`

Implement the `Inputter` interface using macOS CGEvent APIs via CGo. For this task, only implement `Click()` and `MoveMouse()`. Leave other methods (Scroll, Drag, TypeText, KeyCombo) as stubs returning "not yet implemented" — they'll be done in separate tasks.

```go
//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework ApplicationServices -framework Foundation
#include <CoreGraphics/CoreGraphics.h>

// Click at screen coordinates with specified button and click count.
// button: 0=left, 1=right, 2=middle (maps to kCGMouseButton*)
// count: 1=single, 2=double, 3=triple
static int cg_click(float x, float y, int button, int count) {
    CGPoint point = CGPointMake(x, y);

    CGEventType downType, upType;
    CGMouseButton cgButton;

    switch (button) {
        case 1:  // right
            cgButton = kCGMouseButtonRight;
            downType = kCGEventRightMouseDown;
            upType = kCGEventRightMouseUp;
            break;
        case 2:  // middle
            cgButton = kCGMouseButtonCenter;
            downType = kCGEventOtherMouseDown;
            upType = kCGEventOtherMouseUp;
            break;
        default:  // left (0)
            cgButton = kCGMouseButtonLeft;
            downType = kCGEventLeftMouseDown;
            upType = kCGEventLeftMouseUp;
            break;
    }

    for (int i = 0; i < count; i++) {
        CGEventRef down = CGEventCreateMouseEvent(NULL, downType, point, cgButton);
        CGEventRef up = CGEventCreateMouseEvent(NULL, upType, point, cgButton);
        if (!down || !up) {
            if (down) CFRelease(down);
            if (up) CFRelease(up);
            return -1;
        }
        // Set click count for multi-click events
        CGEventSetIntegerValueField(down, kCGMouseEventClickState, i + 1);
        CGEventSetIntegerValueField(up, kCGMouseEventClickState, i + 1);
        CGEventPost(kCGHIDEventTap, down);
        CGEventPost(kCGHIDEventTap, up);
        CFRelease(down);
        CFRelease(up);
    }
    return 0;
}

static int cg_move_mouse(float x, float y) {
    CGPoint point = CGPointMake(x, y);
    CGEventRef move = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, point, kCGMouseButtonLeft);
    if (!move) return -1;
    CGEventPost(kCGHIDEventTap, move);
    CFRelease(move);
    return 0;
}
*/
import "C"

import (
    "fmt"

    "github.com/mj1618/desktop-cli/internal/platform"
)

// DarwinInputter implements the platform.Inputter interface for macOS.
type DarwinInputter struct{}

// NewInputter creates a new macOS inputter.
func NewInputter() *DarwinInputter {
    return &DarwinInputter{}
}

func (inp *DarwinInputter) Click(x, y int, button platform.MouseButton, count int) error {
    if count < 1 {
        count = 1
    }
    cButton := C.int(0)
    switch button {
    case platform.MouseRight:
        cButton = 1
    case platform.MouseMiddle:
        cButton = 2
    }
    if C.cg_click(C.float(x), C.float(y), cButton, C.int(count)) != 0 {
        return fmt.Errorf("failed to click at (%d, %d)", x, y)
    }
    return nil
}

func (inp *DarwinInputter) MoveMouse(x, y int) error {
    if C.cg_move_mouse(C.float(x), C.float(y)) != 0 {
        return fmt.Errorf("failed to move mouse to (%d, %d)", x, y)
    }
    return nil
}

func (inp *DarwinInputter) Scroll(x, y int, dx, dy int) error {
    return fmt.Errorf("scroll not yet implemented")
}

func (inp *DarwinInputter) Drag(fromX, fromY, toX, toY int) error {
    return fmt.Errorf("drag not yet implemented")
}

func (inp *DarwinInputter) TypeText(text string, delayMs int) error {
    return fmt.Errorf("type text not yet implemented")
}

func (inp *DarwinInputter) KeyCombo(keys []string) error {
    return fmt.Errorf("key combo not yet implemented")
}
```

### 2. Register Inputter in Provider — `internal/platform/darwin/init.go`

Update the darwin provider init to also create and register the Inputter:

```go
func init() {
    platform.NewProviderFunc = func() (*platform.Provider, error) {
        reader := NewReader()
        inputter := NewInputter()
        return &platform.Provider{
            Reader:   reader,
            Inputter: inputter,
        }, nil
    }
}
```

### 3. Wire `click` Command — `cmd/click.go`

Replace the `notImplemented("click")` stub with real logic. The click command supports two modes:

**Mode 1: Coordinate click** (`--x` and `--y` provided)
- Click at the specified absolute screen coordinates
- Simple: just call `Inputter.Click(x, y, button, count)`

**Mode 2: Element ID click** (`--id` provided with `--app` and/or `--window`)
- Re-read the element tree for the target app/window
- Find the element with the matching ID
- Compute the center of its bounding box
- Click at that center point

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

    if id > 0 {
        // Element ID mode: re-read and find the element
        if provider.Reader == nil {
            return fmt.Errorf("reader not available on this platform")
        }

        opts := platform.ReadOptions{
            App:    appName,
            Window: window,
        }
        elements, err := provider.Reader.ReadElements(opts)
        if err != nil {
            return err
        }

        elem := findElementByID(elements, id)
        if elem == nil {
            return fmt.Errorf("element with id %d not found", id)
        }

        // Compute center of bounding box
        x = elem.Bounds[0] + elem.Bounds[2]/2
        y = elem.Bounds[1] + elem.Bounds[3]/2
    } else if x == 0 && y == 0 {
        return fmt.Errorf("specify --id or --x/--y coordinates")
    }

    if provider.Inputter == nil {
        return fmt.Errorf("input not available on this platform")
    }

    return provider.Inputter.Click(x, y, button, count)
}

// findElementByID searches the element tree recursively for an element with the given ID.
func findElementByID(elements []model.Element, id int) *model.Element {
    for i := range elements {
        if elements[i].ID == id {
            return &elements[i]
        }
        if found := findElementByID(elements[i].Children, id); found != nil {
            return found
        }
    }
    return nil
}
```

### 4. YAML Success/Error Output

On success, the click command should output a brief confirmation so agents know it succeeded:

```yaml
ok: true
action: click
x: 150
y: 75
button: left
count: 1
```

On error, output the error via the standard Cobra error path (which already outputs to stderr).

Define a small result struct in the click command:

```go
type ClickResult struct {
    OK     bool   `yaml:"ok"`
    Action string `yaml:"action"`
    X      int    `yaml:"x"`
    Y      int    `yaml:"y"`
    Button string `yaml:"button"`
    Count  int    `yaml:"count"`
}
```

After a successful click, print this result using `output.PrintYAML()`.

## Files to Create

- `internal/platform/darwin/inputter.go` — DarwinInputter implementing Inputter interface (Click + MoveMouse via CGEvent, stubs for rest)

## Files to Modify

- `internal/platform/darwin/init.go` — Register DarwinInputter in provider
- `cmd/click.go` — Replace stub with real implementation: coordinate click and element ID click

## Acceptance Criteria

- [ ] `go build ./...` succeeds on macOS
- [ ] `go build ./...` succeeds on non-macOS (Inputter will be nil in provider, click returns "input not available" error)
- [ ] `go test ./...` passes
- [ ] `desktop-cli click --x 100 --y 200` clicks at absolute screen coordinates (100, 200) with the left mouse button
- [ ] `desktop-cli click --x 100 --y 200 --button right` performs a right-click
- [ ] `desktop-cli click --x 100 --y 200 --double` performs a double-click
- [ ] `desktop-cli click --id 5 --app "Finder"` re-reads Finder's element tree, finds element 5, computes center coordinates, and clicks there
- [ ] `desktop-cli click --id 999 --app "Finder"` returns a clear error: "element with id 999 not found"
- [ ] `desktop-cli click` (no flags) returns a clear error: "specify --id or --x/--y coordinates"
- [ ] On success, outputs YAML confirmation with `ok: true`, the coordinates clicked, button, and count
- [ ] macOS accessibility permission error is clear and actionable (for `--id` mode which calls ReadElements)
- [ ] No memory leaks in CGEvent creation (all CGEventRef released)
- [ ] README.md updated with click command examples
- [ ] SKILL.md updated with click command reference

## Implementation Notes

- CGEvent posting requires the same accessibility permission as reading (`AXIsProcessTrusted()`). The permission check already exists in `permissions.go`. For coordinate-mode clicks (`--x --y`), we don't call ReadElements so there's no automatic permission check — but CGEventPost itself will silently fail without permission. Consider adding a permission check to the Inputter as well.
- The `--id` mode is stateless per PLAN.md design: it re-reads the element tree on each click. This means the element's position is current, not stale from a previous read. The trade-off is latency (~200ms extra for the re-read).
- For double-click, CGEvent needs the click count set correctly via `kCGMouseEventClickState`. The C code should send `count` down+up pairs with incrementing click state (1 for first click, 2 for second click in a double-click). The events must be posted in rapid succession (no sleep between them) for the OS to recognize them as a double-click.
- The `findElementByID` helper does a simple recursive search. This is fine for typical element trees (hundreds of elements). No need to optimize.
- The CGo code is inline in the Go file (in the C comment preamble) rather than in separate .c/.h files, since it's small enough. If it grows, extract to separate files.
