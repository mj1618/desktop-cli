//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework ApplicationServices -framework Foundation -framework Carbon
#include <CoreGraphics/CoreGraphics.h>
#include <Carbon/Carbon.h>
#include <unistd.h>

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

// Type a single Unicode character using CGEvent key simulation.
static void cg_type_char(UniChar ch) {
    CGEventRef keyDown = CGEventCreateKeyboardEvent(NULL, 0, true);
    CGEventRef keyUp = CGEventCreateKeyboardEvent(NULL, 0, false);
    CGEventKeyboardSetUnicodeString(keyDown, 1, &ch);
    CGEventKeyboardSetUnicodeString(keyUp, 1, &ch);
    CGEventPost(kCGHIDEventTap, keyDown);
    CGEventPost(kCGHIDEventTap, keyUp);
    CFRelease(keyDown);
    CFRelease(keyUp);
}

// Press a key combo with modifiers.
static void cg_key_combo(CGKeyCode keyCode, CGEventFlags modifiers) {
    CGEventRef keyDown = CGEventCreateKeyboardEvent(NULL, keyCode, true);
    CGEventRef keyUp = CGEventCreateKeyboardEvent(NULL, keyCode, false);
    CGEventSetFlags(keyDown, modifiers);
    CGEventSetFlags(keyUp, modifiers);
    CGEventPost(kCGHIDEventTap, keyDown);
    CGEventPost(kCGHIDEventTap, keyUp);
    CFRelease(keyDown);
    CFRelease(keyUp);
}

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
        if (!drag) {
            // Release mouse to avoid stuck mouse-down state
            CGEventRef upErr = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseUp, pt, kCGMouseButtonLeft);
            if (upErr) {
                CGEventPost(kCGHIDEventTap, upErr);
                CFRelease(upErr);
            }
            return -1;
        }
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
*/
import "C"

import (
	"fmt"
	"strings"
	"time"

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
	// Move mouse to target position first so scroll lands in the right place.
	// Skip if x and y are both 0 (scroll at current mouse position).
	if x != 0 || y != 0 {
		if C.cg_move_mouse(C.float(x), C.float(y)) != 0 {
			return fmt.Errorf("failed to move mouse to (%d, %d) for scroll", x, y)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if C.cg_scroll(C.int(dy), C.int(dx)) != 0 {
		return fmt.Errorf("failed to scroll at (%d, %d)", x, y)
	}
	return nil
}

func (inp *DarwinInputter) Drag(fromX, fromY, toX, toY int) error {
	rc := C.cg_drag(C.float(fromX), C.float(fromY), C.float(toX), C.float(toY), C.int(100))
	if rc != 0 {
		return fmt.Errorf("failed to drag from (%d,%d) to (%d,%d)", fromX, fromY, toX, toY)
	}
	return nil
}

func (inp *DarwinInputter) TypeText(text string, delayMs int) error {
	for _, ch := range text {
		C.cg_type_char(C.UniChar(ch))
		if delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}
	return nil
}

func (inp *DarwinInputter) KeyCombo(keys []string) error {
	keyCode, modifiers, err := parseKeyCombo(keys)
	if err != nil {
		return err
	}
	C.cg_key_combo(C.CGKeyCode(keyCode), C.CGEventFlags(modifiers))
	return nil
}

// macOS virtual key codes from Carbon Events.h.
var keyCodeMap = map[string]uint16{
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

// macOS modifier key flags.
var modifierMap = map[string]uint64{
	"cmd": uint64(C.kCGEventFlagMaskCommand), "command": uint64(C.kCGEventFlagMaskCommand),
	"shift": uint64(C.kCGEventFlagMaskShift),
	"ctrl": uint64(C.kCGEventFlagMaskControl), "control": uint64(C.kCGEventFlagMaskControl),
	"alt": uint64(C.kCGEventFlagMaskAlternate), "opt": uint64(C.kCGEventFlagMaskAlternate), "option": uint64(C.kCGEventFlagMaskAlternate),
}

func parseKeyCombo(keys []string) (C.CGKeyCode, C.CGEventFlags, error) {
	var modifiers uint64
	var keyCode uint16
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
	return C.CGKeyCode(keyCode), C.CGEventFlags(modifiers), nil
}
