# Feature: Implement `scroll` Command with macOS CGEvent Scroll Simulation

## Priority: HIGH (Phase 2, Task 6 — completes the core input action set)

## Problem

Agents using `desktop-cli` can read UI elements and (once click/type land) interact with them, but they cannot scroll. Many desktop workflows require scrolling to reveal content that is off-screen — web pages, long documents, lists, and tables all extend beyond the visible viewport. Without scroll, agents are limited to whatever content fits in the initial view.

The `scroll` command is currently a stub returning "not yet implemented". The `Inputter.Scroll()` method exists in the interface but has no macOS implementation (or it's stubbed as "not yet implemented" in the inputter being built by the click/type task).

## Dependencies

- The `Inputter` backend (`internal/platform/darwin/inputter.go`) is being built by a concurrent task for click/type. This task should either:
  - Wait for the inputter file to exist and add the `Scroll` implementation to it, OR
  - Implement scroll in the same file if it already has a stub `Scroll` method

- The `Provider` registration (init.go) should already include the `Inputter` by the time this task runs.

## What to Build

### 1. Implement `Scroll()` on DarwinInputter — `internal/platform/darwin/inputter.go`

If the `DarwinInputter.Scroll()` method is a stub, replace it with a real implementation using `CGEventCreateScrollWheelEvent`.

If the inputter file doesn't exist yet, create a minimal one with just the Scroll method (and stubs for others), using the same pattern as the concurrent click task.

**CGo scroll implementation:**

```c
// Scroll using CGEventCreateScrollWheelEvent.
// dy: vertical scroll (positive = up, negative = down)
// dx: horizontal scroll (positive = left, negative = right)
static int cg_scroll(int dy, int dx) {
    CGEventRef scroll = CGEventCreateScrollWheelEvent(
        NULL,
        kCGScrollEventUnitLine,
        2,     // number of axes
        dy,    // vertical (line units)
        dx     // horizontal (line units)
    );
    if (!scroll) return -1;
    CGEventPost(kCGHIDEventTap, scroll);
    CFRelease(scroll);
    return 0;
}
```

**Go Scroll method:**

```go
func (inp *DarwinInputter) Scroll(x, y int, dx, dy int) error {
    // Move mouse to the target position first so scroll lands in the right place
    if x != 0 || y != 0 {
        C.cg_move_mouse(C.float(x), C.float(y))
        // Small delay for the mouse move to register
        time.Sleep(10 * time.Millisecond)
    }

    result := C.cg_scroll(C.int(dy), C.int(dx))
    if result != 0 {
        return fmt.Errorf("failed to scroll at (%d, %d)", x, y)
    }
    return nil
}
```

**Direction mapping** — the `--direction` flag maps to `dx`/`dy`:
- `up` → dy = +amount (positive = scroll content up = reveal content above)
- `down` → dy = -amount (negative = scroll content down = reveal content below)
- `left` → dx = +amount
- `right` → dx = -amount

Note: `CGEventCreateScrollWheelEvent` uses "natural" scroll direction where positive dy means scroll up (content moves down). Check the macOS convention and adjust sign accordingly. In practice, on macOS with natural scrolling enabled, positive values scroll content *up* (i.e., the viewport moves down). For predictability, the CLI should use physical direction naming:
- `--direction down` should scroll the viewport downward (reveal content below), which means dy = negative in CGEvent terms.

### 2. Wire `scroll` Command — `cmd/scroll.go`

Replace the `notImplemented("scroll")` stub with real logic:

```go
func runScroll(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }
    if provider.Inputter == nil {
        return fmt.Errorf("input simulation not available on this platform")
    }

    direction, _ := cmd.Flags().GetString("direction")
    amount, _ := cmd.Flags().GetInt("amount")
    x, _ := cmd.Flags().GetInt("x")
    y, _ := cmd.Flags().GetInt("y")
    id, _ := cmd.Flags().GetInt("id")
    appName, _ := cmd.Flags().GetString("app")
    window, _ := cmd.Flags().GetString("window")

    if direction == "" {
        return fmt.Errorf("--direction is required (up, down, left, right)")
    }

    // Validate direction
    var dx, dy int
    switch strings.ToLower(direction) {
    case "up":
        dy = amount
    case "down":
        dy = -amount
    case "left":
        dx = amount
    case "right":
        dx = -amount
    default:
        return fmt.Errorf("invalid direction %q: use up, down, left, or right", direction)
    }

    // If --id specified, resolve element center coordinates
    if id > 0 {
        if appName == "" && window == "" {
            return fmt.Errorf("--id requires --app or --window to scope the element lookup")
        }
        elements, err := provider.Reader.ReadElements(platform.ReadOptions{
            App:    appName,
            Window: window,
        })
        if err != nil {
            return fmt.Errorf("failed to read elements: %w", err)
        }
        elem := findElementByID(elements, id)
        if elem == nil {
            return fmt.Errorf("element with ID %d not found", id)
        }
        x = elem.Bounds[0] + elem.Bounds[2]/2
        y = elem.Bounds[1] + elem.Bounds[3]/2
    }

    return provider.Inputter.Scroll(x, y, dx, dy)
}
```

**Important**: The `findElementByID` helper is shared with `click` and `type` commands. If a `cmd/helpers.go` already exists (created by the click/type task), use it. If not, either create it or define `findElementByID` locally in scroll.go temporarily and refactor later.

### 3. Update scroll command registration

In `cmd/scroll.go`, change the `RunE` from `notImplemented("scroll")` to `runScroll`:

```go
var scrollCmd = &cobra.Command{
    Use:   "scroll",
    Short: "Scroll within a window or element",
    Long:  "Scroll up, down, left, or right within a window or specific element.",
    RunE:  runScroll,
}
```

The flags are already registered in the existing `init()` function — no changes needed there.

### 4. YAML Success Output

On success, output a confirmation so agents know the scroll completed:

```yaml
ok: true
action: scroll
direction: down
amount: 3
x: 500
y: 400
```

Use the same output pattern as other commands (via `output.PrintYAML()` or direct YAML marshal).

## Files to Create

- `cmd/helpers.go` — Shared `findElementByID` helper (only if not already created by click/type task)

## Files to Modify

- `cmd/scroll.go` — Replace stub with real `runScroll()` implementation
- `internal/platform/darwin/inputter.go` — Replace `Scroll()` stub with real CGEvent scroll implementation (if the file exists from click/type task; otherwise create it with Scroll + stubs for others)

## Acceptance Criteria

- [ ] `go build ./...` succeeds on macOS
- [ ] `go build ./...` succeeds on non-macOS (returns "input not available" error)
- [ ] `go test ./...` passes
- [ ] `desktop-cli scroll --direction down` scrolls down 3 lines (default amount) at the current mouse position
- [ ] `desktop-cli scroll --direction up --amount 10` scrolls up 10 lines
- [ ] `desktop-cli scroll --direction left --amount 5` scrolls left 5 units
- [ ] `desktop-cli scroll --direction right` scrolls right 3 units (default)
- [ ] `desktop-cli scroll --direction down --x 500 --y 400` scrolls down at specific coordinates
- [ ] `desktop-cli scroll --direction down --id 6 --app "Safari"` reads element tree, finds element 6, scrolls at its center
- [ ] `desktop-cli scroll --direction down --id 999 --app "Safari"` returns "element not found" error
- [ ] `desktop-cli scroll` (no direction) returns a clear error about --direction being required
- [ ] `desktop-cli scroll --direction diagonal` returns a clear error about valid directions
- [ ] On success, outputs YAML confirmation with action details
- [ ] README.md updated with scroll command examples (already has some — verify accuracy)
- [ ] SKILL.md updated with scroll command reference (already has entry — verify accuracy)

## Implementation Notes

- **CGEvent scroll direction**: `CGEventCreateScrollWheelEvent` with `kCGScrollEventUnitLine` uses line units. Positive values scroll *up* (content moves down), negative values scroll *down* (content moves up). Our `--direction down` should map to negative dy so the viewport moves down to reveal content below.
- **Mouse position matters**: Scroll events are delivered to the window/element under the mouse cursor. Moving the mouse to the target coordinates before scrolling ensures the scroll lands in the right place.
- **Default amount**: 3 scroll lines is a reasonable default — enough to move meaningfully without overshooting.
- **Element scrolling**: When `--id` is used, the mouse moves to the element's center before scrolling. This works for scroll areas, lists, web content areas, etc. The AX API also supports `AXUIElementPerformAction` with scroll actions, but CGEvent scroll is simpler and more reliable for this use case.
- **No sleep between scroll lines**: Unlike some tools that scroll one line at a time with delays, `CGEventCreateScrollWheelEvent` handles the amount natively. A single event with `amount=5` scrolls 5 lines at once.
- **Imports needed**: `strings` for direction parsing, `fmt` for errors. Also `time` if adding a small delay after mouse move.
- **Coordinate (0,0)**: When no --x/--y or --id is specified, x and y default to 0. In this case, skip the mouse move and scroll at the current mouse position. Check for this condition: only call `cg_move_mouse` when x != 0 or y != 0, or when --id was used.

---

## Completion Notes (Agent 70d9cf04 / Task ab0cb372)

### What was implemented:

1. **CGo scroll function** — Added `cg_scroll(int dy, int dx)` static C function to `internal/platform/darwin/inputter.go` CGo preamble. Uses `CGEventCreateScrollWheelEvent` with `kCGScrollEventUnitLine` for 2-axis scrolling.

2. **DarwinInputter.Scroll()** — Replaced the stub with a real implementation that:
   - Moves the mouse to target (x, y) if coordinates are non-zero (with 10ms delay for registration)
   - Calls `cg_scroll(dy, dx)` to perform the actual scroll event
   - Scrolls at current mouse position when x=0, y=0

3. **scroll command** (`cmd/scroll.go`) — Fully wired with:
   - Direction parsing: up/down/left/right mapped to dy/dx values
   - `--amount` flag (default 3 lines)
   - `--x`/`--y` for coordinate-targeted scrolling
   - `--id` with `--app`/`--window` for element-targeted scrolling (uses `findElementByID` from `cmd/helpers.go`)
   - YAML success output via `ScrollResult` struct and `output.PrintYAML()`

4. **Documentation** — Updated README.md with scroll usage section and SKILL.md with expanded scroll examples.

### Files modified:
- `internal/platform/darwin/inputter.go` — Added `cg_scroll` C function, replaced `Scroll()` stub
- `cmd/scroll.go` — Full rewrite from stub to working command
- `README.md` — Added scroll section in Usage
- `SKILL.md` — Expanded scroll examples

### Build/Test Status:
- `go build ./...` passes
- `go test ./...` passes (all packages)
- `go vet ./...` passes
- Runtime testing requires a real macOS desktop with accessibility permissions
