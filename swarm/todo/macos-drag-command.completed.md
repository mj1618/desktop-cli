# Feature: Implement `drag` Command with macOS CGEvent Drag Simulation

## Priority: MEDIUM (Phase 4 — enables drag-and-drop workflows for agents)

## Problem

Agents using `desktop-cli` can click, type, scroll, and focus — but cannot drag elements. The `drag` command is currently a stub returning "not yet implemented" at both the command level (`cmd/drag.go`) and the platform level (`inputter.go:Drag()`). Without drag support, agents cannot:

- Move files between folders in Finder
- Drag items in Kanban boards or task managers
- Resize windows or panes by dragging splitters
- Interact with sliders and range inputs
- Rearrange tabs or list items
- Draw selections in design tools

## What to Build

### 1. C Drag Function — `internal/platform/darwin/inputter.go` (inline CGo block)

Add a `cg_drag` function to the existing CGo block in `inputter.go`, following the same pattern as `cg_click`, `cg_scroll`, etc.

```c
// Drag from (fromX,fromY) to (toX,toY) using left mouse button.
// duration_ms: time for the drag animation in milliseconds (0 = instant).
// Steps are interpolated linearly between start and end points.
static int cg_drag(float fromX, float fromY, float toX, float toY, int duration_ms) {
    CGPoint startPoint = CGPointMake(fromX, fromY);
    CGPoint endPoint = CGPointMake(toX, toY);

    // 1. Move mouse to start position
    CGEventRef move = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, startPoint, kCGMouseButtonLeft);
    if (!move) return -1;
    CGEventPost(kCGHIDEventTap, move);
    CFRelease(move);

    // Small delay to ensure move registers
    usleep(10000); // 10ms

    // 2. Mouse down at start position
    CGEventRef down = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseDown, startPoint, kCGMouseButtonLeft);
    if (!down) return -1;
    CGEventPost(kCGHIDEventTap, down);
    CFRelease(down);

    // 3. Interpolate drag path with multiple dragged events
    int steps = 20;
    if (duration_ms <= 0) {
        duration_ms = 100; // default 100ms drag animation
    }
    int delay_per_step = (duration_ms * 1000) / steps; // microseconds

    for (int i = 1; i <= steps; i++) {
        float t = (float)i / (float)steps;
        float x = fromX + (toX - fromX) * t;
        float y = fromY + (toY - fromY) * t;
        CGPoint pt = CGPointMake(x, y);

        CGEventRef drag = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseDragged, pt, kCGMouseButtonLeft);
        if (!drag) return -1;
        CGEventPost(kCGHIDEventTap, drag);
        CFRelease(drag);

        usleep(delay_per_step);
    }

    // 4. Mouse up at end position
    CGEventRef up = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseUp, endPoint, kCGMouseButtonLeft);
    if (!up) return -1;
    CGEventPost(kCGHIDEventTap, up);
    CFRelease(up);

    return 0;
}
```

**Key points:**
- Uses `kCGEventLeftMouseDragged` (not `kCGEventMouseMoved`) for the intermediate events — this is critical for macOS to recognize it as a drag gesture
- Interpolates in 20 steps with configurable total duration for smooth, realistic drag motion
- Default duration of 100ms is fast enough for automation but slow enough for macOS to register the drag
- Includes `usleep` between steps so the OS event queue processes each movement
- Add `#include <unistd.h>` at the top of the CGo block for `usleep`

### 2. Implement Go `Drag()` Method — `internal/platform/darwin/inputter.go`

Replace the stub `Drag()` method:

```go
func (inp *DarwinInputter) Drag(fromX, fromY, toX, toY int) error {
    rc := C.cg_drag(C.float(fromX), C.float(fromY), C.float(toX), C.float(toY), C.int(100))
    if rc != 0 {
        return fmt.Errorf("failed to drag from (%d,%d) to (%d,%d)", fromX, fromY, toX, toY)
    }
    return nil
}
```

### 3. Wire Command Logic — `cmd/drag.go`

Replace the stub `notImplemented("drag")` with a real `runDrag` function. Follow the same patterns used by `click.go` and `scroll.go`:

```go
package cmd

import (
    "fmt"

    "github.com/mj1618/desktop-cli/internal/output"
    "github.com/mj1618/desktop-cli/internal/platform"
    "github.com/spf13/cobra"
)

// DragResult is the YAML output of a successful drag.
type DragResult struct {
    OK     bool   `yaml:"ok"`
    Action string `yaml:"action"`
    FromX  int    `yaml:"from_x"`
    FromY  int    `yaml:"from_y"`
    ToX    int    `yaml:"to_x"`
    ToY    int    `yaml:"to_y"`
}

var dragCmd = &cobra.Command{
    Use:   "drag",
    Short: "Drag from one point to another",
    Long:  "Drag from one point to another using coordinates or element IDs.",
    RunE:  runDrag,
}

func init() {
    rootCmd.AddCommand(dragCmd)
    dragCmd.Flags().Int("from-x", 0, "Start X coordinate")
    dragCmd.Flags().Int("from-y", 0, "Start Y coordinate")
    dragCmd.Flags().Int("to-x", 0, "End X coordinate")
    dragCmd.Flags().Int("to-y", 0, "End Y coordinate")
    dragCmd.Flags().Int("from-id", 0, "Start element (center)")
    dragCmd.Flags().Int("to-id", 0, "End element (center)")
    dragCmd.Flags().String("app", "", "Scope to application")
    dragCmd.Flags().String("window", "", "Scope to window")
}

func runDrag(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }
    if provider.Inputter == nil {
        return fmt.Errorf("input simulation not available on this platform")
    }

    fromX, _ := cmd.Flags().GetInt("from-x")
    fromY, _ := cmd.Flags().GetInt("from-y")
    toX, _ := cmd.Flags().GetInt("to-x")
    toY, _ := cmd.Flags().GetInt("to-y")
    fromID, _ := cmd.Flags().GetInt("from-id")
    toID, _ := cmd.Flags().GetInt("to-id")
    appName, _ := cmd.Flags().GetString("app")
    window, _ := cmd.Flags().GetString("window")

    // Resolve element IDs to coordinates if specified
    if fromID > 0 || toID > 0 {
        if provider.Reader == nil {
            return fmt.Errorf("reader not available on this platform")
        }
        elements, err := provider.Reader.ReadElements(platform.ReadOptions{
            App:    appName,
            Window: window,
        })
        if err != nil {
            return fmt.Errorf("failed to read elements: %w", err)
        }

        if fromID > 0 {
            elem := findElementByID(elements, fromID)
            if elem == nil {
                return fmt.Errorf("from-element with ID %d not found", fromID)
            }
            fromX = elem.Bounds[0] + elem.Bounds[2]/2
            fromY = elem.Bounds[1] + elem.Bounds[3]/2
        }

        if toID > 0 {
            elem := findElementByID(elements, toID)
            if elem == nil {
                return fmt.Errorf("to-element with ID %d not found", toID)
            }
            toX = elem.Bounds[0] + elem.Bounds[2]/2
            toY = elem.Bounds[1] + elem.Bounds[3]/2
        }
    }

    // Validate that we have coordinates
    if fromX == 0 && fromY == 0 && toX == 0 && toY == 0 {
        return fmt.Errorf("specify --from-x/--from-y and --to-x/--to-y or --from-id/--to-id")
    }

    if err := provider.Inputter.Drag(fromX, fromY, toX, toY); err != nil {
        return err
    }

    return output.PrintYAML(DragResult{
        OK:     true,
        Action: "drag",
        FromX:  fromX,
        FromY:  fromY,
        ToX:    toX,
        ToY:    toY,
    })
}
```

### 4. Update Documentation

**README.md** — Add a "Drag" section after "Scroll":

```markdown
### Drag

```bash
# Drag from one screen coordinate to another
desktop-cli drag --from-x 100 --from-y 200 --to-x 400 --to-y 300

# Drag between elements by ID
desktop-cli drag --from-id 3 --to-id 7 --app "Finder"

# Mix: drag from element to coordinates
desktop-cli drag --from-id 5 --to-x 500 --to-y 600 --app "Finder"
```
```

**SKILL.md** — Add drag examples to the quick reference:

```markdown
### Drag

```bash
desktop-cli drag --from-x 100 --from-y 200 --to-x 400 --to-y 300
desktop-cli drag --from-id 3 --to-id 7 --app "Finder"
```
```

## Files to Modify

- `internal/platform/darwin/inputter.go` — Add `cg_drag` C function to CGo block; replace stub `Drag()` method with real implementation
- `cmd/drag.go` — Replace stub with full `runDrag` implementation including element ID resolution, coordinate validation, and YAML output
- `README.md` — Add drag command usage section
- `SKILL.md` — Add drag command to quick reference

## Files NOT to Create

No new files needed — all changes go into existing files.

## Acceptance Criteria

- [ ] `go build ./...` succeeds on macOS
- [ ] `go test ./...` passes
- [ ] `desktop-cli drag --from-x 100 --from-y 100 --to-x 300 --to-y 300` drags from (100,100) to (300,300)
- [ ] `desktop-cli drag --from-id 3 --to-id 7 --app "Finder"` resolves element IDs and drags between their centers
- [ ] `desktop-cli drag --from-id 5 --to-x 500 --to-y 600 --app "Finder"` supports mixing element ID and coordinates
- [ ] Command outputs YAML result with `ok: true`, action, and coordinates
- [ ] Error messages are clear when IDs are not found or coordinates are missing
- [ ] Drag animation is smooth enough for macOS to recognize as a proper drag gesture (not just a teleport)
- [ ] README.md and SKILL.md are updated with drag examples

## Implementation Notes

- **CGEvent drag events**: macOS requires `kCGEventLeftMouseDragged` events between mouse-down and mouse-up. Simply posting mouse-down at point A and mouse-up at point B does NOT work as a drag — the intermediate dragged events are essential for macOS to recognize the gesture.
- **Step interpolation**: 20 steps over ~100ms provides smooth motion. Too few steps or too fast can cause macOS to miss the drag intent. Too many steps or too slow makes automation sluggish.
- **`usleep` for pacing**: The C function uses `usleep()` between drag steps. Add `#include <unistd.h>` to the CGo includes if not already present.
- **Element ID resolution**: Uses the same `findElementByID` helper and `ReadElements` pattern as `click.go` and `scroll.go`. When either `--from-id` or `--to-id` is used, we read the element tree once and resolve both.
- **Coordinate (0,0) edge case**: The validation checks if ALL four coordinates are zero. This means dragging FROM (0,0) TO somewhere is technically valid if the destination isn't (0,0). This matches the click command's behavior where (0,0) means "not specified."
- **No `--duration` flag**: The drag duration is hardcoded at 100ms in the platform layer. If agents need slower drags (e.g., for precision in design tools), a `--duration` flag could be added later — but keep it simple for now.
- **Existing interface**: The `platform.Inputter` interface already defines `Drag(fromX, fromY, toX, toY int) error`, so no interface changes are needed.
