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

// Lazily-initialised event source for keyboard events.
// Uses kCGEventSourceStatePrivate so that synthetic events never inherit
// stale modifier flags from the combined session state (e.g. a previous
// Cmd+L would otherwise taint subsequent TypeText characters with the
// Command modifier, turning "t" into Cmd+T).
static CGEventSourceRef _kbSource = NULL;
static CGEventSourceRef get_kb_source() {
    if (_kbSource == NULL) {
        _kbSource = CGEventSourceCreate(kCGEventSourceStatePrivate);
    }
    return _kbSource;
}

// Type a single Unicode character using CGEvent key simulation.
// keyCode should be the real macOS virtual key code for the character
// (use 0 only when the correct code is unknown, e.g. non-ASCII).
// modifiers should include kCGEventFlagMaskShift when the character
// requires the Shift key (e.g. uppercase letters, symbols like '!').
static int cg_type_char(UniChar ch, CGKeyCode keyCode, CGEventFlags modifiers) {
    CGEventSourceRef src = get_kb_source();
    CGEventRef keyDown = CGEventCreateKeyboardEvent(src, keyCode, true);
    CGEventRef keyUp   = CGEventCreateKeyboardEvent(src, keyCode, false);
    if (!keyDown || !keyUp) {
        if (keyDown) CFRelease(keyDown);
        if (keyUp)   CFRelease(keyUp);
        return -1;
    }
    CGEventKeyboardSetUnicodeString(keyDown, 1, &ch);
    CGEventKeyboardSetUnicodeString(keyUp, 1, &ch);
    CGEventSetFlags(keyDown, modifiers);
    CGEventSetFlags(keyUp, modifiers);
    CGEventPost(kCGHIDEventTap, keyDown);
    usleep(1000); // 1 ms between key-down and key-up
    CGEventPost(kCGHIDEventTap, keyUp);
    CFRelease(keyDown);
    CFRelease(keyUp);
    return 0;
}

// Press a key combo with modifiers.
static int cg_key_combo(CGKeyCode keyCode, CGEventFlags modifiers) {
    CGEventSourceRef src = get_kb_source();
    CGEventRef keyDown = CGEventCreateKeyboardEvent(src, keyCode, true);
    CGEventRef keyUp = CGEventCreateKeyboardEvent(src, keyCode, false);
    if (!keyDown || !keyUp) {
        if (keyDown) CFRelease(keyDown);
        if (keyUp)   CFRelease(keyUp);
        return -1;
    }
    CGEventSetFlags(keyDown, modifiers);
    CGEventSetFlags(keyUp, modifiers);
    CGEventPost(kCGHIDEventTap, keyDown);
    usleep(1000); // 1 ms between key-down and key-up
    CGEventPost(kCGHIDEventTap, keyUp);
    CFRelease(keyDown);
    CFRelease(keyUp);
    return 0;
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
	// Minimum inter-character delay to prevent event loss.
	if delayMs < 5 {
		delayMs = 5
	}
	for _, ch := range text {
		keyCode, modifiers := charToKeyCode(ch)
		if C.cg_type_char(C.UniChar(ch), C.CGKeyCode(keyCode), C.CGEventFlags(modifiers)) != 0 {
			return fmt.Errorf("failed to type character %q", string(ch))
		}
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}
	return nil
}

func (inp *DarwinInputter) KeyCombo(keys []string) error {
	keyCode, modifiers, err := parseKeyCombo(keys)
	if err != nil {
		return err
	}
	if C.cg_key_combo(C.CGKeyCode(keyCode), C.CGEventFlags(modifiers)) != 0 {
		return fmt.Errorf("failed to post key combo")
	}
	return nil
}

// charKeyInfo holds the virtual key code and shift state for a character.
type charKeyInfo struct {
	keyCode uint16
	shift   bool
}

// charKeyMap maps ASCII characters to their macOS virtual key codes (US keyboard layout).
// Using correct key codes makes synthetic events indistinguishable from real
// keyboard input, which is required by apps like Chrome's omnibox that inspect
// raw key codes rather than only the Unicode string on the event.
var charKeyMap = map[rune]charKeyInfo{
	// Lowercase letters
	'a': {0x00, false}, 'b': {0x0B, false}, 'c': {0x08, false}, 'd': {0x02, false},
	'e': {0x0E, false}, 'f': {0x03, false}, 'g': {0x05, false}, 'h': {0x04, false},
	'i': {0x22, false}, 'j': {0x26, false}, 'k': {0x28, false}, 'l': {0x25, false},
	'm': {0x2E, false}, 'n': {0x2D, false}, 'o': {0x1F, false}, 'p': {0x23, false},
	'q': {0x0C, false}, 'r': {0x0F, false}, 's': {0x01, false}, 't': {0x11, false},
	'u': {0x20, false}, 'v': {0x09, false}, 'w': {0x0D, false}, 'x': {0x07, false},
	'y': {0x10, false}, 'z': {0x06, false},
	// Uppercase letters (same key codes, with Shift)
	'A': {0x00, true}, 'B': {0x0B, true}, 'C': {0x08, true}, 'D': {0x02, true},
	'E': {0x0E, true}, 'F': {0x03, true}, 'G': {0x05, true}, 'H': {0x04, true},
	'I': {0x22, true}, 'J': {0x26, true}, 'K': {0x28, true}, 'L': {0x25, true},
	'M': {0x2E, true}, 'N': {0x2D, true}, 'O': {0x1F, true}, 'P': {0x23, true},
	'Q': {0x0C, true}, 'R': {0x0F, true}, 'S': {0x01, true}, 'T': {0x11, true},
	'U': {0x20, true}, 'V': {0x09, true}, 'W': {0x0D, true}, 'X': {0x07, true},
	'Y': {0x10, true}, 'Z': {0x06, true},
	// Digits
	'0': {0x1D, false}, '1': {0x12, false}, '2': {0x13, false}, '3': {0x14, false},
	'4': {0x15, false}, '5': {0x17, false}, '6': {0x16, false}, '7': {0x1A, false},
	'8': {0x1C, false}, '9': {0x19, false},
	// Symbols (unshifted keys, US layout)
	'-': {0x1B, false}, '=': {0x18, false}, '[': {0x21, false}, ']': {0x1E, false},
	'\\': {0x2A, false}, ';': {0x29, false}, '\'': {0x27, false}, '`': {0x32, false},
	',': {0x2B, false}, '.': {0x2F, false}, '/': {0x2C, false},
	// Symbols (shifted keys, US layout)
	'!': {0x12, true}, '@': {0x13, true}, '#': {0x14, true}, '$': {0x15, true},
	'%': {0x17, true}, '^': {0x16, true}, '&': {0x1A, true}, '*': {0x1C, true},
	'(': {0x19, true}, ')': {0x1D, true}, '_': {0x1B, true}, '+': {0x18, true},
	'{': {0x21, true}, '}': {0x1E, true}, '|': {0x2A, true}, ':': {0x29, true},
	'"': {0x27, true}, '~': {0x32, true}, '<': {0x2B, true}, '>': {0x2F, true},
	'?': {0x2C, true},
	// Whitespace
	' ': {0x31, false}, '\t': {0x30, false}, '\n': {0x24, false},
}

// charToKeyCode returns the virtual key code and CGEvent modifier flags
// for a character.  For unknown characters it returns key code 0 with no
// modifiers, falling back to the Unicode-string-only approach.
func charToKeyCode(ch rune) (uint16, uint64) {
	if info, ok := charKeyMap[ch]; ok {
		var mods uint64
		if info.shift {
			mods = uint64(C.kCGEventFlagMaskShift)
		}
		return info.keyCode, mods
	}
	return 0, 0
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
