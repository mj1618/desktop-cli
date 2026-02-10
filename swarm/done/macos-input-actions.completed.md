# Feature: Implement `click`, `type`, and `focus` Commands with macOS Input Simulation

## Priority: HIGH (Phase 2 — enables the core agent read-act loop)

## Problem

The `list` and `read` commands allow agents to *see* what's on screen, but agents cannot *act* on anything. The `click`, `type`, and `focus` commands are all stubs returning "not yet implemented". Without these, agents cannot complete any real desktop automation workflow.

These three commands share a common dependency: the macOS input simulation backend (`Inputter` and `WindowManager` interfaces). Building them together is the most efficient approach since they share the same CGo infrastructure.

## What to Build

### 1. macOS Input Simulator — `internal/platform/darwin/inputter.go`

Implement the `platform.Inputter` interface using macOS `CGEvent` APIs via CGo.

```go
//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ApplicationServices -framework Carbon
#include <CoreGraphics/CoreGraphics.h>
#include <Carbon/Carbon.h>

// Click at coordinates with specified button and click count
void cg_click(int x, int y, int button, int count);

// Move mouse to coordinates
void cg_move_mouse(int x, int y);

// Scroll at coordinates
void cg_scroll(int x, int y, int dx, int dy);

// Drag from one point to another
void cg_drag(int fromX, int fromY, int toX, int toY);

// Type a UTF-8 string using CGEvent key simulation
void cg_type_text(const char* text);

// Press a key combination (modifier flags + key code)
void cg_key_combo(int keyCode, int modifiers);
*/
import "C"

type DarwinInputter struct{}

func NewInputter() *DarwinInputter {
    return &DarwinInputter{}
}
```

#### C Implementation — `internal/platform/darwin/input_sim.c` + `input_sim.h`

The C layer handles all CGEvent creation and posting:

**Mouse click** (`cg_click`):
1. Move mouse to (x, y) with `CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, point, 0)`
2. For each click in `count`:
   - Create mouse-down event: `CGEventCreateMouseEvent(NULL, kCGEventLeftMouseDown, point, kCGMouseButtonLeft)` (or Right/Other for other buttons)
   - Set click count: `CGEventSetIntegerValueField(event, kCGMouseEventClickState, clickNum)`
   - Post event: `CGEventPost(kCGHIDEventTap, event)`
   - Create mouse-up event similarly
   - Post mouse-up event
   - Release both events

**Mouse button mapping**:
- `button == 0` → left: `kCGEventLeftMouseDown` / `kCGEventLeftMouseUp`, `kCGMouseButtonLeft`
- `button == 1` → right: `kCGEventRightMouseDown` / `kCGEventRightMouseUp`, `kCGMouseButtonRight`
- `button == 2` → middle: `kCGEventOtherMouseDown` / `kCGEventOtherMouseUp`, `kCGMouseButtonCenter`

**Type text** (`cg_type_text`):
- For each Unicode character in the text string:
  - Create a keyboard event: `CGEventCreateKeyboardEvent(NULL, 0, true)`
  - Set the Unicode string on it: `CGEventKeyboardSetUnicodeString(event, 1, &unichar)`
  - Post key-down and key-up events
  - Release events
- This approach handles all Unicode characters without needing a keycode lookup table for regular text.

**Key combo** (`cg_key_combo`):
- Create keyboard event with the correct virtual keycode: `CGEventCreateKeyboardEvent(NULL, keyCode, true)`
- Set modifier flags: `CGEventSetFlags(event, flags)`
  - `kCGEventFlagMaskCommand` for Cmd
  - `kCGEventFlagMaskShift` for Shift
  - `kCGEventFlagMaskAlternate` for Option/Alt
  - `kCGEventFlagMaskControl` for Control
- Post key-down, then key-up with same modifiers
- Release events

**Virtual keycode mapping** (needed for key combos — define in C or Go):

Common keycodes (macOS virtual keycodes from `<Carbon/Carbon.h>` `Events.h`):
```
enter/return = 36    tab = 48         space = 49
delete/backspace = 51   escape = 53
up = 126    down = 125    left = 123    right = 124
home = 115  end = 119   pageup = 116  pagedown = 121
a=0 s=1 d=2 f=3 h=4 g=5 z=6 x=7 c=8 v=9 b=11 q=12 w=13 e=14 r=15
t=17 y=16 u=32 i=34 o=31 p=35
```

**Go key string parsing** — in `inputter.go`, parse the `--key` flag value:
1. Split on `+` to get parts
2. Identify modifiers: `cmd`/`command`, `ctrl`/`control`, `shift`, `alt`/`opt`/`option`
3. The last part is the key name
4. Map the key name to a virtual keycode
5. Combine modifier flags
6. Call `cg_key_combo(keyCode, modifierFlags)`

**Scroll** (`cg_scroll`):
- Move mouse to (x, y) first
- `CGEventCreateScrollWheelEvent(NULL, kCGScrollEventUnitLine, 2, dy, dx)`
- Post event, release

**Drag** (`cg_drag`):
- Move mouse to (fromX, fromY)
- Create mouse-down event at from point, post
- Create mouse-dragged events incrementally from (fromX, fromY) to (toX, toY), post each
- Create mouse-up event at to point, post
- Small sleep between events (e.g., 5ms) for the system to register the drag

### 2. macOS Window Manager — `internal/platform/darwin/window_manager.go`

Implement `platform.WindowManager` interface using NSRunningApplication and AX APIs.

```go
//go:build darwin

package darwin

type DarwinWindowManager struct{}

func NewWindowManager() *DarwinWindowManager {
    return &DarwinWindowManager{}
}
```

**FocusWindow** logic:
1. If `--app` is specified: Use `NSRunningApplication` to find the app by name and call `activateWithOptions:` via CGo/ObjC
2. If `--pid` is specified: Create `NSRunningApplication` from PID and activate
3. If `--window` or `--window-id` is specified: First activate the app (find PID from window list), then use AX API to raise the specific window:
   - `AXUIElementCreateApplication(pid)`
   - Get windows attribute
   - Find matching window
   - `AXUIElementPerformAction(window, kAXRaiseAction)`
   - `AXUIElementSetAttributeValue(window, kAXMainAttribute, kCFBooleanTrue)`

**GetFrontmostApp**:
- Use `NSWorkspace.sharedWorkspace.frontmostApplication` to get name and PID
- Or reuse the existing `cg_get_frontmost_pid()` from `window_list.c` and extend it

#### C Implementation — `internal/platform/darwin/window_focus.c` + `window_focus.h`

```c
#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>

// Activate app by PID (bring to foreground)
int ns_activate_app(pid_t pid);

// Raise a specific window by AX API
int ax_raise_window(pid_t pid, int windowIndex);

// Get frontmost app name and PID
int ns_get_frontmost_app(char** outName, pid_t* outPid);
```

### 3. Wire Click Command — `cmd/click.go`

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

    clickCount := 1
    if double {
        clickCount = 2
    }

    // If --id is specified, resolve coordinates by re-reading the element tree
    if id != 0 {
        if appName == "" && window == "" {
            return fmt.Errorf("--id requires --app or --window to scope the element lookup")
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
            return fmt.Errorf("element with ID %d not found", id)
        }
        // Click center of element bounds
        x = elem.Bounds[0] + elem.Bounds[2]/2
        y = elem.Bounds[1] + elem.Bounds[3]/2
    } else if x == 0 && y == 0 {
        return fmt.Errorf("specify --id with --app/--window, or --x and --y coordinates")
    }

    return provider.Inputter.Click(x, y, button, clickCount)
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

### 4. Wire Type Command — `cmd/typecmd.go`

Replace the stub with real logic:

```go
func runType(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }

    text, _ := cmd.Flags().GetString("text")
    key, _ := cmd.Flags().GetString("key")
    delay, _ := cmd.Flags().GetInt("delay")
    id, _ := cmd.Flags().GetInt("id")
    appName, _ := cmd.Flags().GetString("app")
    window, _ := cmd.Flags().GetString("window")

    // Get text from positional arg if not from flag
    if text == "" && len(args) > 0 {
        text = args[0]
    }

    if text == "" && key == "" {
        return fmt.Errorf("specify --text, --key, or a positional text argument")
    }

    // If --id is specified, click the element first to focus it
    if id != 0 {
        if appName == "" && window == "" {
            return fmt.Errorf("--id requires --app or --window to scope the element lookup")
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
            return fmt.Errorf("element with ID %d not found", id)
        }
        // Click center to focus
        cx := elem.Bounds[0] + elem.Bounds[2]/2
        cy := elem.Bounds[1] + elem.Bounds[3]/2
        if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
            return fmt.Errorf("failed to focus element %d: %w", id, err)
        }
        // Small delay to let focus settle
        time.Sleep(50 * time.Millisecond)
    }

    // Type text or press key combo
    if key != "" {
        keys := strings.Split(key, "+")
        return provider.Inputter.KeyCombo(keys)
    }

    return provider.Inputter.TypeText(text, delay)
}
```

Note: `findElementByID` is shared between click and type — consider putting it in a `cmd/helpers.go` or in `model` package.

### 5. Wire Focus Command — `cmd/focus.go`

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
        return fmt.Errorf("specify --app, --window, --window-id, or --pid")
    }

    return provider.WindowManager.FocusWindow(platform.FocusOptions{
        App:      appName,
        Window:   window,
        WindowID: windowID,
        PID:      pid,
    })
}
```

### 6. Update Provider Registration — `internal/platform/darwin/init.go`

Add `Inputter` and `WindowManager` to the provider:

```go
func init() {
    platform.NewProviderFunc = func() (*platform.Provider, error) {
        reader := NewReader()
        inputter := NewInputter()
        windowManager := NewWindowManager()
        return &platform.Provider{
            Reader:        reader,
            Inputter:      inputter,
            WindowManager: windowManager,
        }, nil
    }
}
```

### 7. Shared Helper — `cmd/helpers.go`

Extract `findElementByID` into a shared helper since both `click` and `type` need it:

```go
package cmd

import "github.com/mj1618/desktop-cli/internal/model"

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

## Files to Create

- `internal/platform/darwin/inputter.go` — Go `DarwinInputter` implementing `platform.Inputter` interface
- `internal/platform/darwin/input_sim.c` — C implementation of CGEvent mouse/keyboard/scroll simulation
- `internal/platform/darwin/input_sim.h` — C header for input simulation functions
- `internal/platform/darwin/window_manager.go` — Go `DarwinWindowManager` implementing `platform.WindowManager` interface
- `internal/platform/darwin/window_focus.c` — C implementation of app activation and window raising
- `internal/platform/darwin/window_focus.h` — C header for window focus functions
- `cmd/helpers.go` — Shared helpers like `findElementByID`

## Files to Modify

- `internal/platform/darwin/init.go` — Register `Inputter` and `WindowManager` in provider
- `cmd/click.go` — Replace stub with real `runClick()` implementation
- `cmd/typecmd.go` — Replace stub with real `runType()` implementation
- `cmd/focus.go` — Replace stub with real `runFocus()` implementation
- `README.md` — Update if needed with new command examples
- `SKILL.md` — Update if needed with new command reference

## Acceptance Criteria

- [ ] `go build ./...` succeeds on macOS
- [ ] `go build ./...` succeeds on non-macOS (Inputter/WindowManager nil in provider, commands return helpful errors)
- [ ] `go test ./...` passes
- [ ] `desktop-cli click --x 100 --y 200` clicks at absolute coordinates
- [ ] `desktop-cli click --x 100 --y 200 --button right` right-clicks
- [ ] `desktop-cli click --x 100 --y 200 --double` double-clicks
- [ ] `desktop-cli click --id 5 --app "Finder"` re-reads Finder's element tree and clicks element 5's center
- [ ] `desktop-cli click` with no args returns a helpful error
- [ ] `desktop-cli click --id 5` without --app returns helpful error about needing --app or --window
- [ ] `desktop-cli type --text "hello world"` types the text
- [ ] `desktop-cli type "hello world"` types the text (positional arg)
- [ ] `desktop-cli type --key "cmd+c"` sends Cmd+C key combo
- [ ] `desktop-cli type --key "enter"` sends Enter
- [ ] `desktop-cli type --key "tab"` sends Tab
- [ ] `desktop-cli type --key "cmd+shift+t"` sends Cmd+Shift+T
- [ ] `desktop-cli type --id 5 --app "Finder" --text "hello"` clicks element 5 first then types
- [ ] `desktop-cli type` with no args returns a helpful error
- [ ] `desktop-cli focus --app "Finder"` brings Finder to the foreground
- [ ] `desktop-cli focus --pid <pid>` brings app with that PID to foreground
- [ ] `desktop-cli focus --window "Downloads"` finds and raises the matching window
- [ ] `desktop-cli focus` with no args returns a helpful error
- [ ] Accessibility permission errors are clear and actionable
- [ ] README.md and SKILL.md updated if needed

## Implementation Notes

- **Permission requirements**: `click`, `type`, and `focus` all require Accessibility permission. Check with `AXIsProcessTrusted()` before attempting operations. The permission check already exists in `darwin/permissions.go`.
- **CGEvent posting**: Events must be posted to `kCGHIDEventTap` for them to be system-wide. Use `CGEventPost(kCGHIDEventTap, event)`.
- **Thread safety**: CGEvent functions are thread-safe, but rapid event posting may need small sleeps (1-5ms) between events for the system to process them reliably.
- **Element ID resolution** for `click --id` and `type --id`: This re-reads the element tree each time. This is stateless and matches the PLAN.md design. The `read` command implementation (being built separately) provides the `ReadElements` method this depends on. If `read` isn't yet functional, `click --id` and `type --id` won't work but `click --x --y` and `type --text` will work independently.
- **Key combo parsing**: The `+` separator in `--key "cmd+c"` means we split on `+`. All parts except the last are modifiers. The last part is the key. Support common aliases: `cmd`/`command`, `ctrl`/`control`, `alt`/`opt`/`option`, `shift`.
- **Carbon framework**: Needed for virtual keycode constants (`kVK_*`). Include `<Carbon/Carbon.h>` in the CGo preamble.
- **Unicode text typing**: Use `CGEventKeyboardSetUnicodeString` for typing arbitrary text rather than mapping each character to a keycode. This is simpler and handles non-ASCII characters.
- **Focus before type with --id**: When `type --id` is used, click the element center to focus it before typing. Add a small delay (50ms) between the click and the typing to let focus settle.
- **Drag can be deferred**: The `drag` and `scroll` commands can be done in a follow-up task since they're less critical for the core agent loop. However, the `Inputter.Scroll` and `Inputter.Drag` methods should still be implemented in the backend since they share the same C infrastructure — just the command wiring can be separate if needed.
