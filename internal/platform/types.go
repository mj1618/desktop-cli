package platform

import (
	"fmt"
	"strconv"
	"strings"
)

// MouseButton represents a mouse button.
type MouseButton int

const (
	MouseLeft MouseButton = iota
	MouseRight
	MouseMiddle
)

// ParseMouseButton converts a string flag value to MouseButton.
func ParseMouseButton(s string) (MouseButton, error) {
	switch strings.ToLower(s) {
	case "left":
		return MouseLeft, nil
	case "right":
		return MouseRight, nil
	case "middle":
		return MouseMiddle, nil
	default:
		return MouseLeft, fmt.Errorf("unknown mouse button: %q (expected left, right, or middle)", s)
	}
}

// Bounds represents a screen rectangle.
type Bounds struct {
	X, Y, Width, Height int
}

// ParseBBox parses a "x,y,w,h" string into a Bounds.
func ParseBBox(s string) (*Bounds, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid bbox %q: expected x,y,w,h", s)
	}
	vals := make([]int, 4)
	for i, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, fmt.Errorf("invalid bbox %q: %w", s, err)
		}
		vals[i] = v
	}
	return &Bounds{X: vals[0], Y: vals[1], Width: vals[2], Height: vals[3]}, nil
}

// ReadOptions controls what elements to read and how to filter them.
type ReadOptions struct {
	App         string   // Filter by application name
	Window      string   // Filter by window title substring
	WindowID    int      // Filter by system window ID (0 = unset)
	PID         int      // Filter by process ID (0 = unset)
	Depth       int      // Max traversal depth (0 = unlimited)
	Roles       []string // Only include these roles (empty = all)
	VisibleOnly bool     // Only include visible elements
	BBox        *Bounds  // Only include elements within this bounding box (nil = no filter)
	Compact     bool     // Use compact output format
}

// ListOptions controls window/app listing.
type ListOptions struct {
	Apps bool   // List applications instead of windows
	PID  int    // Filter by PID
	App  string // Filter by app name
}

// FocusOptions specifies what to focus.
type FocusOptions struct {
	App      string
	Window   string
	WindowID int
	PID      int
}

// ScreenshotOptions configures what to capture.
type ScreenshotOptions struct {
	App      string  // Capture frontmost window of this app
	Window   string  // Capture window matching this title substring
	WindowID int     // Capture window by system ID
	PID      int     // Capture frontmost window of this PID
	Format   string  // "png" or "jpg"
	Quality  int     // JPEG quality 1-100 (ignored for PNG)
	Scale    float64 // Scale factor 0.1-1.0 (default 0.5)
}

// ActionOptions configures which element to act on and what action to perform.
type ActionOptions struct {
	App      string // Scope to application
	Window   string // Scope to window
	WindowID int    // Scope to window by system ID
	PID      int    // Scope to process
	ID       int    // Element ID (from read output)
	Action   string // Action to perform: "press", "cancel", "pick", "increment", "decrement", "confirm", "showMenu", "raise"
}
