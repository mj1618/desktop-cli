//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ImageIO -framework AppKit
#include "screenshot.h"
#include <stdlib.h>
*/
import "C"
import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"strings"
	"unsafe"

	"github.com/mj1618/desktop-cli/internal/platform"
)

// CheckScreenRecordingPermission checks if the process has macOS screen recording permission.
func CheckScreenRecordingPermission() error {
	if C.cg_check_screen_recording() == 0 {
		return fmt.Errorf(
			"screen recording permission required\n\n" +
				"Grant permission at: System Settings > Privacy & Security > Screen Recording\n" +
				"Add your terminal app (e.g. Terminal.app, iTerm2, or the IDE running this command).\n" +
				"Then restart the terminal and try again.")
	}
	return nil
}

// RequestScreenRecordingPermission triggers the macOS screen recording permission prompt
// if permission has not yet been granted. Returns true if already granted.
func RequestScreenRecordingPermission() bool {
	return C.cg_request_screen_recording() != 0
}

// DarwinScreenshotter implements platform.Screenshotter for macOS.
type DarwinScreenshotter struct {
	reader *DarwinReader
}

// NewScreenshotter creates a new macOS screenshotter.
func NewScreenshotter(reader *DarwinReader) *DarwinScreenshotter {
	return &DarwinScreenshotter{reader: reader}
}

// CaptureWindow captures a screenshot of a specific window or the full screen.
func (s *DarwinScreenshotter) CaptureWindow(opts platform.ScreenshotOptions) ([]byte, error) {
	if err := CheckScreenRecordingPermission(); err != nil {
		return nil, err
	}

	// Resolve the target window ID
	windowID := opts.WindowID
	if windowID == 0 && (opts.App != "" || opts.Window != "" || opts.PID != 0) {
		var err error
		windowID, err = s.resolveWindowID(opts)
		if err != nil {
			return nil, err
		}
	}

	// Set defaults
	scale := opts.Scale
	if scale <= 0 || scale > 1.0 {
		scale = 0.5
	}
	format := 0 // PNG
	if opts.Format == "jpg" || opts.Format == "jpeg" {
		format = 1
	}
	quality := opts.Quality
	if quality <= 0 || quality > 100 {
		quality = 80
	}

	// If including menu bar with a window capture, composite both images
	if opts.IncludeMenuBar && windowID != 0 {
		return s.captureWindowWithMenuBar(windowID, format, quality, scale)
	}

	var result C.ScreenshotResult
	var rc C.int

	if windowID != 0 {
		rc = C.cg_capture_window(C.int(windowID), C.int(format), C.int(quality),
			C.float(scale), &result)
	} else {
		rc = C.cg_capture_screen(C.int(format), C.int(quality),
			C.float(scale), &result)
	}

	if rc != 0 {
		return nil, fmt.Errorf("screenshot capture failed (check Screen Recording permission in System Settings > Privacy & Security > Screen Recording)")
	}
	defer C.cg_free_screenshot(&result)

	return C.GoBytes(unsafe.Pointer(result.data), C.int(result.length)), nil
}

// captureWindowWithMenuBar captures a window and the menu bar, compositing them
// into a single image with the menu bar on top.
func (s *DarwinScreenshotter) captureWindowWithMenuBar(windowID, format, quality int, scale float64) ([]byte, error) {
	// Capture the menu bar region (top of main display, full width)
	menuBarHeight := float64(C.cg_get_menubar_height())
	displayWidth := float64(C.cg_get_display_width())

	var menuResult C.ScreenshotResult
	rc := C.cg_capture_rect(0, 0, C.float(displayWidth), C.float(menuBarHeight),
		C.int(format), C.int(quality), C.float(scale), &menuResult)
	if rc != 0 {
		return nil, fmt.Errorf("failed to capture menu bar")
	}
	menuData := C.GoBytes(unsafe.Pointer(menuResult.data), C.int(menuResult.length))
	C.cg_free_screenshot(&menuResult)

	// Capture the window
	var winResult C.ScreenshotResult
	rc = C.cg_capture_window(C.int(windowID), C.int(format), C.int(quality),
		C.float(scale), &winResult)
	if rc != 0 {
		return nil, fmt.Errorf("screenshot capture failed (check Screen Recording permission)")
	}
	winData := C.GoBytes(unsafe.Pointer(winResult.data), C.int(winResult.length))
	C.cg_free_screenshot(&winResult)

	// Decode both images
	menuImg, err := decodeImage(menuData, format)
	if err != nil {
		return nil, fmt.Errorf("failed to decode menu bar image: %w", err)
	}
	winImg, err := decodeImage(winData, format)
	if err != nil {
		return nil, fmt.Errorf("failed to decode window image: %w", err)
	}

	// Composite: menu bar on top, window below
	menuBounds := menuImg.Bounds()
	winBounds := winImg.Bounds()
	compositeWidth := menuBounds.Dx()
	if winBounds.Dx() > compositeWidth {
		compositeWidth = winBounds.Dx()
	}
	compositeHeight := menuBounds.Dy() + winBounds.Dy()

	composite := image.NewRGBA(image.Rect(0, 0, compositeWidth, compositeHeight))
	draw.Draw(composite, image.Rect(0, 0, menuBounds.Dx(), menuBounds.Dy()),
		menuImg, menuBounds.Min, draw.Src)
	draw.Draw(composite, image.Rect(0, menuBounds.Dy(), winBounds.Dx()+0, menuBounds.Dy()+winBounds.Dy()),
		winImg, winBounds.Min, draw.Src)

	// Encode the composite image
	return encodeImage(composite, format, quality)
}

func decodeImage(data []byte, format int) (image.Image, error) {
	if format == 1 {
		return jpeg.Decode(bytes.NewReader(data))
	}
	return png.Decode(bytes.NewReader(data))
}

func encodeImage(img image.Image, format, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if format == 1 {
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, err
		}
	} else {
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// resolveWindowID finds the window ID matching the given options.
func (s *DarwinScreenshotter) resolveWindowID(opts platform.ScreenshotOptions) (int, error) {
	listOpts := platform.ListOptions{}
	if opts.App != "" {
		listOpts.App = opts.App
	}
	if opts.PID != 0 {
		listOpts.PID = opts.PID
	}

	windows, err := s.reader.ListWindows(listOpts)
	if err != nil {
		return 0, fmt.Errorf("failed to list windows: %w", err)
	}
	if len(windows) == 0 {
		return 0, fmt.Errorf("no windows found matching the specified criteria")
	}

	// If a window title filter is specified, find the matching window
	if opts.Window != "" {
		for _, w := range windows {
			if strings.Contains(strings.ToLower(w.Title), strings.ToLower(opts.Window)) {
				return w.ID, nil
			}
		}
		return 0, fmt.Errorf("no window found matching title %q", opts.Window)
	}

	// Return the first matching window
	return windows[0].ID, nil
}
