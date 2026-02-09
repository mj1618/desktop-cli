package platform

import "github.com/mj1618/desktop-cli/internal/model"

// Reader reads the UI element tree from the OS accessibility layer.
type Reader interface {
	// ReadElements returns the element tree for the specified target.
	ReadElements(opts ReadOptions) ([]model.Element, error)

	// ListWindows returns all windows, optionally filtered.
	ListWindows(opts ListOptions) ([]model.Window, error)
}

// Inputter simulates mouse and keyboard input.
type Inputter interface {
	Click(x, y int, button MouseButton, count int) error
	MoveMouse(x, y int) error
	Scroll(x, y int, dx, dy int) error
	Drag(fromX, fromY, toX, toY int) error
	TypeText(text string, delayMs int) error
	KeyCombo(keys []string) error
}

// WindowManager manages window focus and positioning.
type WindowManager interface {
	FocusWindow(opts FocusOptions) error
	GetFrontmostApp() (string, int, error)
}

// Screenshotter captures screenshots.
type Screenshotter interface {
	// CaptureWindow captures a screenshot of a specific window or the full screen.
	// Returns the image bytes in the requested format.
	CaptureWindow(opts ScreenshotOptions) ([]byte, error)
}

// ActionPerformer performs accessibility actions directly on UI elements.
type ActionPerformer interface {
	// PerformAction executes an accessibility action on an element identified
	// by its sequential ID within the given read scope.
	PerformAction(opts ActionOptions) error
}
