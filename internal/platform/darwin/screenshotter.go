//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ImageIO
#include "screenshot.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
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
