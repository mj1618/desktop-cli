# Feature: Implement `click` and `type` Commands with macOS CGEvent Input Simulation

## Priority: HIGH (Phase 2 — completes the read→act agent loop)

## Problem

The tool can read UI elements (`list`, `read`) but cannot interact with them. The `click` and `type` commands are stubs returning "not yet implemented". The `Inputter` interface has no macOS implementation. Without click and type, agents cannot complete any desktop automation workflow — they can see buttons but can't press them.

## What to Build

### 1. macOS Inputter Backend — `internal/platform/darwin/inputter.go`

Implement the `platform.Inputter` interface using macOS CGEvent APIs via CGo.

#### CGo Preamble

```go
//go:build darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework Carbon

#include <CoreGraphics/CoreGraphics.h>
#include <Carbon/Carbon.h>

// Click at screen coordinates. button: 0=left, 1=right, 2=middle. count: 1=single, 2=double
static inline void cg_click(float x, float y, int button, int count) {
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
        default: // left
            downType = kCGEventLeftMouseDown;
            upType = kCGEventLeftMouseUp;
            cgButton = kCGMouseButtonLeft;
            break;
    }

    CGEventRef down = CGEventCreateMouseEvent(NULL, downType, point, cgButton);
    CGEventRef up = CGEventCreateMouseEvent(NULL, upType, point, cgButton);

    if (count == 2) {
        CGEventSetIntegerValueField(down, kCGMouseEventClickState, 2);
        CGEventSetIntegerValueField(up, kCGMouseEventClickState, 2);
    }

    // For double click, send click pair twice with click state
    if (count == 2) {
        CGEventRef down1 = CGEventCreateMouseEvent(NULL, downType, point, cgButton);
        CGEventRef up1 = CGEventCreateMouseEvent(NULL, upType, point, cgButton);
        CGEventSetIntegerValueField(down1, kCGMouseEventClickState, 1);
        CGEventSetIntegerValueField(up1, kCGMouseEventClickState, 1);
        CGEventPost(kCGHIDEventTap, down1);
        CGEventPost(kCGHIDEventTap, up1);
        CFRelease(down1);
        CFRelease(up1);
    }

    CGEventPost(kCGHIDEventTap, down);
    CGEventPost(kCGHIDEventTap, up);
    CFRelease(down);
    CFRelease(up);
}

static inline void cg_move_mouse(float x, float y) {
    CGPoint point = CGPointMake(x, y);
    CGEventRef move = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, point, kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, move);
    CFRelease(move);
}

static inline void cg_scroll(int dx, int dy) {
    CGEventRef scroll = CGEventCreateScrollWheelEvent(NULL, kCGScrollEventUnitLine, 2, dy, dx);
    CGEventPost(kCGHIDEventTap, scroll);
    CFRelease(scroll);
}

static inline void cg_drag(float fromX, float fromY, float toX, float toY) {
    CGPoint from = CGPointMake(fromX, fromY);
    CGPoint to = CGPointMake(toX, toY);

    CGEventRef down = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseDown, from, kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, down);
    CFRelease(down);

    CGEventRef drag = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseDragged, to, kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, drag);
    CFRelease(drag);

    CGEventRef up = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseUp, to, kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, up);
    CFRelease(up);
}

// Type a single Unicode character using CGEvent key simulation
static inline void cg_type_char(UniChar ch) {
    CGEventRef keyDown = CGEventCreateKeyboardEvent(NULL, 0, true);
    CGEventRef keyUp = CGEventCreateKeyboardEvent(NULL, 0, false);
    CGEventKeyboardSetUnicodeString(keyDown, 1, &ch);
    CGEventKeyboardSetUnicodeString(keyUp, 1, &ch);
    CGEventPost(kCGHIDEventTap, keyDown);
    CGEventPost(kCGHIDEventTap, keyUp);
    CFRelease(keyDown);
    CFRelease(keyUp);
}

// Press a key combo with modifiers. keyCode is a virtual key code, modifiers is CGEventFlags.
static inline void cg_key_combo(CGKeyCode keyCode, CGEventFlags modifiers) {
    CGEventRef keyDown = CGEventCreateKeyboardEvent(NULL, keyCode, true);
    CGEventRef keyUp = CGEventCreateKeyboardEvent(NULL, keyCode, false);
    CGEventSetFlags(keyDown, modifiers);
    CGEventSetFlags(keyUp, modifiers);
    CGEventPost(kCGHIDEventTap, keyDown);
    CGEventPost(kCGHIDEventTap, keyUp);
    CFRelease(keyDown);
    CFRelease(keyUp);
}
*/
import "C"
```

#### Go Implementation

```go
type DarwinInputter struct{}

func NewInputter() *DarwinInputter {
    return &DarwinInputter{}
}

func (d *DarwinInputter) Click(x, y int, button platform.MouseButton, count int) error {
    btnInt := 0 // left
    switch button {
    case platform.MouseRight:
        btnInt = 1
    case platform.MouseMiddle:
        btnInt = 2
    }
    C.cg_click(C.float(x), C.float(y), C.int(btnInt), C.int(count))
    return nil
}

func (d *DarwinInputter) MoveMouse(x, y int) error {
    C.cg_move_mouse(C.float(x), C.float(y))
    return nil
}

func (d *DarwinInputter) Scroll(x, y int, dx, dy int) error {
    // Move mouse to position first, then scroll
    C.cg_move_mouse(C.float(x), C.float(y))
    C.cg_scroll(C.int(dx), C.int(dy))
    return nil
}

func (d *DarwinInputter) Drag(fromX, fromY, toX, toY int) error {
    C.cg_drag(C.float(fromX), C.float(fromY), C.float(toX), C.float(toY))
    return nil
}

func (d *DarwinInputter) TypeText(text string, delayMs int) error {
    for _, ch := range text {
        C.cg_type_char(C.UniChar(ch))
        if delayMs > 0 {
            time.Sleep(time.Duration(delayMs) * time.Millisecond)
        }
    }
    return nil
}

func (d *DarwinInputter) KeyCombo(keys []string) error {
    keyCode, modifiers, err := parseKeyCombo(keys)
    if err != nil {
        return err
    }
    C.cg_key_combo(C.CGKeyCode(keyCode), C.CGEventFlags(modifiers))
    return nil
}
```

#### Key Code Mapping — `parseKeyCombo` helper

Must map common key names to macOS virtual key codes (`Carbon/Events.h` constants):

```go
var keyCodeMap = map[string]C.CGKeyCode{
    "a": 0x00, "b": 0x0B, "c": 0x08, "d": 0x02, "e": 0x0E, "f": 0x03,
    "g": 0x05, "h": 0x04, "i": 0x22, "j": 0x26, "k": 0x28, "l": 0x25,
    "m": 0x2E, "n": 0x2D, "o": 0x1F, "p": 0x23, "q": 0x0C, "r": 0x0F,
    "s": 0x01, "t": 0x11, "u": 0x20, "v": 0x09, "w": 0x0D, "x": 0x07,
    "y": 0x10, "z": 0x06,
    "0": 0x1D, "1": 0x12, "2": 0x13, "3": 0x14, "4": 0x15,
    "5": 0x17, "6": 0x16, "7": 0x1A, "8": 0x1C, "9": 0x19,
    "return": 0x24, "enter": 0x24, "tab": 0x30, "space": 0x31,
    "delete": 0x33, "backspace": 0x33, "escape": 0x35, "esc": 0x35,
    "up": 0x7E, "down": 0x7D, "left": 0x7B, "right": 0x7C,
    "home": 0x73, "end": 0x77, "pageup": 0x74, "pagedown": 0x79,
    "f1": 0x7A, "f2": 0x78, "f3": 0x63, "f4": 0x76, "f5": 0x60,
    "f6": 0x61, "f7": 0x62, "f8": 0x64, "f9": 0x65, "f10": 0x6D,
    "f11": 0x67, "f12": 0x6F,
}

var modifierMap = map[string]C.CGEventFlags{
    "cmd":     C.CGEventFlags(C.kCGEventFlagMaskCommand),
    "command": C.CGEventFlags(C.kCGEventFlagMaskCommand),
    "shift":   C.CGEventFlags(C.kCGEventFlagMaskShift),
    "ctrl":    C.CGEventFlags(C.kCGEventFlagMaskControl),
    "control": C.CGEventFlags(C.kCGEventFlagMaskControl),
    "alt":     C.CGEventFlags(C.kCGEventFlagMaskAlternate),
    "opt":     C.CGEventFlags(C.kCGEventFlagMaskAlternate),
    "option":  C.CGEventFlags(C.kCGEventFlagMaskAlternate),
}

func parseKeyCombo(keys []string) (C.CGKeyCode, C.CGEventFlags, error) {
    var modifiers C.CGEventFlags
    var keyCode C.CGKeyCode
    found := false

    for _, k := range keys {
        k = strings.ToLower(strings.TrimSpace(k))
        if mod, ok := modifierMap[k]; ok {
            modifiers |= mod
        } else if code, ok := keyCodeMap[k]; ok {
            keyCode = code
            found = true
        } else {
            return 0, 0, fmt.Errorf("unknown key: %q", k)
        }
    }
    if !found {
        return 0, 0, fmt.Errorf("no key specified in combo, only modifiers")
    }
    return keyCode, modifiers, nil
}
```

### 2. Register Inputter in Provider — Modify `internal/platform/darwin/init.go`

Update the init function to also create and register the DarwinInputter:

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

### 3. Wire `click` Command — Modify `cmd/click.go`

Replace the `notImplemented("click")` stub with real logic:

```go
func runClick(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }
    if provider.Inputter == nil {
        return fmt.Errorf("input simulation not available on this platform")
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

    // If --id is specified, resolve element coordinates via read
    if id > 0 {
        if appName == "" {
            return fmt.Errorf("--app is required when using --id")
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

        // Click center of element bounds
        x = elem.Bounds[0] + elem.Bounds[2]/2
        y = elem.Bounds[1] + elem.Bounds[3]/2
    } else if x == 0 && y == 0 {
        return fmt.Errorf("specify --id or --x/--y coordinates")
    }

    return provider.Inputter.Click(x, y, button, count)
}

// findElementByID searches the element tree recursively for a given ID.
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

### 4. Wire `type` Command — Modify `cmd/typecmd.go`

Replace the `notImplemented("type")` stub:

```go
func runType(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }
    if provider.Inputter == nil {
        return fmt.Errorf("input simulation not available on this platform")
    }

    text, _ := cmd.Flags().GetString("text")
    key, _ := cmd.Flags().GetString("key")
    delayMs, _ := cmd.Flags().GetInt("delay")
    id, _ := cmd.Flags().GetInt("id")
    appName, _ := cmd.Flags().GetString("app")
    window, _ := cmd.Flags().GetString("window")

    // Positional arg overrides --text flag
    if len(args) > 0 {
        text = args[0]
    }

    // If --id specified, click the element first to focus it
    if id > 0 {
        if appName == "" {
            return fmt.Errorf("--app is required when using --id")
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
        cx := elem.Bounds[0] + elem.Bounds[2]/2
        cy := elem.Bounds[1] + elem.Bounds[3]/2
        if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
            return fmt.Errorf("failed to focus element: %w", err)
        }
        time.Sleep(50 * time.Millisecond) // Brief pause for focus to take effect
    }

    if key != "" {
        // Parse key combo like "cmd+c" or "ctrl+shift+t"
        keys := strings.Split(key, "+")
        return provider.Inputter.KeyCombo(keys)
    }

    if text != "" {
        return provider.Inputter.TypeText(text, delayMs)
    }

    return fmt.Errorf("specify --text, --key, or a positional text argument")
}
```

### 5. Shared helper: `findElementByID`

Put `findElementByID` in a shared location (e.g., `cmd/helpers.go` or inline in each command file) since both `click` and `type` need it. A small helper file is cleaner:

```go
// cmd/helpers.go
package cmd

import "github.com/mj1618/desktop-cli/internal/model"

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

- `internal/platform/darwin/inputter.go` — macOS CGEvent input simulator implementing `platform.Inputter`
- `cmd/helpers.go` — Shared `findElementByID` helper used by click and type commands

## Files to Modify

- `internal/platform/darwin/init.go` — Register `DarwinInputter` in the provider
- `cmd/click.go` — Replace stub with `runClick()` that supports `--id`, `--x/--y`, `--button`, `--double`
- `cmd/typecmd.go` — Replace stub with `runType()` that supports `--text`, `--key`, `--delay`, `--id`
- `README.md` — Update if needed (click and type usage examples already present)
- `SKILL.md` — Update if needed (click and type examples already present)

## Acceptance Criteria

- [ ] `go build ./...` succeeds on macOS
- [ ] `go test ./...` passes
- [ ] `desktop-cli click --x 500 --y 500` clicks at absolute coordinates
- [ ] `desktop-cli click --x 500 --y 500 --button right` right-clicks
- [ ] `desktop-cli click --x 500 --y 500 --double` double-clicks
- [ ] `desktop-cli click --id 3 --app "Finder"` reads Finder's element tree, finds element 3, and clicks its center
- [ ] `desktop-cli click` with no args returns a helpful error
- [ ] `desktop-cli click --id 999 --app "Finder"` returns "element not found" error
- [ ] `desktop-cli click --id 3` without `--app` returns an error explaining --app is required
- [ ] `desktop-cli type --text "hello"` types "hello" character by character
- [ ] `desktop-cli type "hello"` (positional arg) types "hello"
- [ ] `desktop-cli type --key "cmd+c"` sends Cmd+C
- [ ] `desktop-cli type --key "ctrl+shift+t"` sends Ctrl+Shift+T
- [ ] `desktop-cli type --key "enter"` presses Enter
- [ ] `desktop-cli type --key "tab"` presses Tab
- [ ] `desktop-cli type --text "hello" --delay 50` types with 50ms delay between characters
- [ ] `desktop-cli type --id 4 --app "Safari" --text "search query"` clicks element 4 first, then types
- [ ] `desktop-cli type` with no args returns a helpful error
- [ ] Key combo parsing handles modifiers: cmd, command, shift, ctrl, control, alt, opt, option
- [ ] Key combo parsing handles keys: a-z, 0-9, return/enter, tab, space, delete/backspace, escape/esc, arrow keys, F1-F12, home, end, pageup, pagedown
- [ ] Unknown key names produce a clear error message
- [ ] No memory leaks in C code (all CGEvent objects released with CFRelease)
- [ ] Provider.Inputter is non-nil on macOS after NewProvider()

## Implementation Notes

- **CGEvent API** is the correct macOS API for synthetic input. It requires Accessibility permissions (same as already checked for read operations).
- **Unicode typing** uses `CGEventKeyboardSetUnicodeString` which handles any character without needing virtual key code lookup. This is simpler and more correct than mapping characters to key codes.
- **Key combos** need virtual key codes from Carbon `Events.h`. The key code values are stable macOS constants.
- **Click via --id** is stateless as designed in PLAN.md: re-reads the element tree, finds the element, computes center, clicks. No daemon needed.
- **The 50ms sleep** after clicking to focus an element (in `type --id`) prevents race conditions where the OS hasn't processed the click focus before keystrokes arrive.
- **Import `time`** in `cmd/typecmd.go` for the sleep.
- **Import `strings`** in `cmd/typecmd.go` for key combo parsing.
- **Import `fmt`** in both `cmd/click.go` and `cmd/typecmd.go` for error formatting.
- The `findElementByID` helper must search recursively through children since elements are in a tree structure.
