# Feature: Implement `drag` Command with macOS CGEvent Drag Simulation

## Priority: HIGH (Phase 4 — completes the input action set for agents)

## Problem

The `drag` command is currently a stub that returns "not yet implemented". The `Inputter.Drag()` method in `internal/platform/darwin/inputter.go:165` also returns an error. Without drag, agents cannot perform essential desktop interactions like:

- Moving files between folders in Finder
- Rearranging items in lists or kanban boards
- Resizing windows or UI panels
- Selecting text by dragging
- Drawing or interacting with canvas-based UIs

All other input commands (click, type, scroll, focus) are fully implemented. Drag is the last missing input action.

## What to Build

### 1. CGo Drag Implementation — `internal/platform/darwin/inputter.go`

Add a C function `cg_drag` in the existing CGo block in `inputter.go` (after the existing `cg_scroll` function, before the closing `*/`). Also add `#include <unistd.h>` to the existing includes for `usleep()`:

```c
#include <unistd.h>

// Drag from (fromX, fromY) to (toX, toY) using left mouse button.
// Uses mouse-down at start, mouse-dragged events along the path, and mouse-up at end.
// duration_ms: total drag duration in milliseconds (0 = default ~100ms).
static int cg_drag(float fromX, float fromY, float toX, float toY, int duration_ms) {
    CGPoint from = CGPointMake(fromX, fromY);
    CGPoint to = CGPointMake(toX, toY);

    // 1. Move mouse to start position
    CGEventRef move = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, from, kCGMouseButtonLeft);
    if (!move) return -1;
    CGEventPost(kCGHIDEventTap, move);
    CFRelease(move);

    // Small delay to ensure move registers
    usleep(50000); // 50ms

    // 2. Mouse down at start
    CGEventRef down = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseDown, from, kCGMouseButtonLeft);
    if (!down) return -1;
    CGEventPost(kCGHIDEventTap, down);
    CFRelease(down);

    // 3. Generate intermediate drag events along the path
    // Use at least 10 steps for smooth dragging that apps recognize
    int steps = 10;
    if (duration_ms > 0) {
        steps = duration_ms / 10; // one step per 10ms
        if (steps < 10) steps = 10;
        if (steps > 200) steps = 200;
    }

    int step_delay_us = 10000; // 10ms default
    if (duration_ms > 0) {
        step_delay_us = (duration_ms * 1000) / steps;
    }

    for (int i = 1; i <= steps; i++) {
        float t = (float)i / (float)steps;
        float cx = fromX + (toX - fromX) * t;
        float cy = fromY + (toY - fromY) * t;
        CGPoint current = CGPointMake(cx, cy);

        CGEventRef drag = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseDragged, current, kCGMouseButtonLeft);
        if (!drag) continue;
        CGEventPost(kCGHIDEventTap, drag);
        CFRelease(drag);
        usleep(step_delay_us);
    }

    // 4. Mouse up at destination
    CGEventRef up = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseUp, to, kCGMouseButtonLeft);
    if (!up) return -1;
    CGEventPost(kCGHIDEventTap, up);
    CFRelease(up);

    return 0;
}
```

Key implementation details:
- **Intermediate drag events are essential**: Many macOS apps (Finder, browsers) ignore a simple "mouse-down at A, mouse-up at B" sequence. They require `kCGEventLeftMouseDragged` events along the path to recognize a drag operation.
- **Timing matters**: A small delay between events ensures the system and target app process each event. 10ms per step is a good default.
- **`usleep` for delays**: Use `usleep()` (microseconds) in the C layer for precise timing without CGo overhead.

### 2. Update Go `Drag()` Method — `internal/platform/darwin/inputter.go`

Replace the stub `Drag()` method at line 165-167:

**Before:**
```go
func (inp *DarwinInputter) Drag(fromX, fromY, toX, toY int) error {
	return fmt.Errorf("drag not yet implemented")
}
```

**After:**
```go
func (inp *DarwinInputter) Drag(fromX, fromY, toX, toY int) error {
	rc := C.cg_drag(C.float(fromX), C.float(fromY), C.float(toX), C.float(toY), C.int(0))
	if rc != 0 {
		return fmt.Errorf("failed to drag from (%d,%d) to (%d,%d)", fromX, fromY, toX, toY)
	}
	return nil
}
```

### 3. Wire Drag Command — `cmd/drag.go`

Replace the entire file. Change `RunE` from `notImplemented("drag")` to `runDrag`, add imports for `output` and `platform` packages, and add the `DragResult` struct and `runDrag` function.

The command supports two modes:

**Coordinate mode**: `desktop-cli drag --from-x 100 --from-y 200 --to-x 300 --to-y 400`

**Element ID mode**: `desktop-cli drag --from-id 5 --to-id 12 --app "Finder"`
- Re-reads the element tree (like `click --id` does)
- Resolves element IDs to center coordinates of their bounding boxes using existing `findElementByID()` from `helpers.go`
- Then performs the drag between those coordinates

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
		opts := platform.ReadOptions{
			App:    appName,
			Window: window,
		}
		elements, err := provider.Reader.ReadElements(opts)
		if err != nil {
			return err
		}

		if fromID > 0 {
			elem := findElementByID(elements, fromID)
			if elem == nil {
				return fmt.Errorf("element with id %d not found", fromID)
			}
			fromX = elem.Bounds[0] + elem.Bounds[2]/2
			fromY = elem.Bounds[1] + elem.Bounds[3]/2
		}
		if toID > 0 {
			elem := findElementByID(elements, toID)
			if elem == nil {
				return fmt.Errorf("element with id %d not found", toID)
			}
			toX = elem.Bounds[0] + elem.Bounds[2]/2
			toY = elem.Bounds[1] + elem.Bounds[3]/2
		}
	}

	// Validate that we have coordinates
	if fromX == 0 && fromY == 0 {
		return fmt.Errorf("specify --from-id or --from-x/--from-y coordinates")
	}
	if toX == 0 && toY == 0 {
		return fmt.Errorf("specify --to-id or --to-x/--to-y coordinates")
	}

	if provider.Inputter == nil {
		return fmt.Errorf("input not available on this platform")
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

**README.md** — Add a "Drag" section after "Scroll" (before "Focus a window"):

```markdown
### Drag

\```bash
# Drag between absolute coordinates
desktop-cli drag --from-x 100 --from-y 200 --to-x 300 --to-y 400

# Drag from one element to another by ID
desktop-cli drag --from-id 5 --to-id 12 --app "Finder"

# Mix: drag from an element to coordinates
desktop-cli drag --from-id 5 --to-x 500 --to-y 300 --app "Finder"
\```
```

**SKILL.md** — Add drag to the quick reference.

## Files to Modify

- `internal/platform/darwin/inputter.go` — Add `cg_drag` C function in CGo block, add `#include <unistd.h>`, replace stub `Drag()` with real implementation
- `cmd/drag.go` — Replace stub with full `runDrag()` implementation, add `DragResult` struct, add imports
- `README.md` — Add drag usage examples
- `SKILL.md` — Add drag to quick reference

## No New Files Needed

All changes fit into existing files.

## Acceptance Criteria

- [ ] `go build ./...` succeeds on macOS
- [ ] `go build ./...` succeeds on non-macOS (drag returns platform-not-supported error gracefully)
- [ ] `go test ./...` passes
- [ ] `desktop-cli drag --from-x 100 --from-y 200 --to-x 300 --to-y 400` performs a visible drag and outputs YAML confirmation
- [ ] `desktop-cli drag --from-id 5 --to-id 12 --app "Finder"` resolves element IDs and drags between them
- [ ] `desktop-cli drag --from-id 5 --to-x 500 --to-y 300 --app "Finder"` mixes element ID and coordinate modes
- [ ] Error message when no from/to coordinates or IDs are specified
- [ ] Error message when element IDs are not found
- [ ] README.md and SKILL.md are updated with drag examples

## Implementation Notes

- **Follow click.go patterns**: The drag command mirrors how `click.go` resolves element IDs — read the tree, find the element via `findElementByID()` from `helpers.go`, compute center coordinates from bounding box. Output uses `output.PrintYAML()` for consistent YAML output.
- **CGo drag events**: The critical insight is that macOS apps need `kCGEventLeftMouseDragged` events (not just mouse-moved) between mouse-down and mouse-up. Without these intermediate events, most apps won't recognize the drag gesture.
- **Smooth path**: Generate at least 10 intermediate points along the drag path. This ensures apps like Finder detect the drag gesture. Fewer points may be silently ignored.
- **Timing**: ~10ms between drag events is a reliable default. Too fast and events may be coalesced/dropped; too slow and the drag feels unresponsive. The total default drag takes ~100ms.
- **`usleep` in C**: Using `usleep()` in the C layer avoids CGo call overhead per step. The entire drag operation happens in a single CGo call, keeping it fast.
- **Left button only**: The PLAN.md spec and `platform.Inputter` interface only define `Drag(fromX, fromY, toX, toY int)` — no button parameter. Left-button drag is the standard. Right-button drag can be added later if needed.
- **No duration flag on the command (yet)**: The C function accepts `duration_ms` but the Go `Drag()` interface doesn't expose it. Passing 0 uses the default ~100ms total drag time. A `--duration` flag could be added later.
