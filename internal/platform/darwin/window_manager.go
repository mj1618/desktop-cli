//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework ApplicationServices -framework CoreFoundation -framework Foundation
#include "window_focus.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/mj1618/desktop-cli/internal/platform"
)

// DarwinWindowManager implements the platform.WindowManager interface for macOS.
type DarwinWindowManager struct {
	reader *DarwinReader
}

// NewWindowManager creates a new macOS window manager.
func NewWindowManager(reader *DarwinReader) *DarwinWindowManager {
	return &DarwinWindowManager{reader: reader}
}

func (wm *DarwinWindowManager) FocusWindow(opts platform.FocusOptions) error {
	if err := CheckAccessibilityPermission(); err != nil {
		return err
	}

	pid := opts.PID

	// Resolve PID from --app if needed
	if pid == 0 && opts.App != "" {
		windows, err := wm.reader.ListWindows(platform.ListOptions{App: opts.App})
		if err != nil {
			return fmt.Errorf("failed to find app %q: %w", opts.App, err)
		}
		if len(windows) == 0 {
			return fmt.Errorf("no windows found for app %q", opts.App)
		}
		pid = windows[0].PID
	}

	// Resolve PID from --window-id
	if pid == 0 && opts.WindowID > 0 {
		windows, err := wm.reader.ListWindows(platform.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list windows: %w", err)
		}
		for _, w := range windows {
			if w.ID == opts.WindowID {
				pid = w.PID
				break
			}
		}
		if pid == 0 {
			return fmt.Errorf("no window found with ID %d", opts.WindowID)
		}
	}

	// Resolve PID from --window (title match)
	if pid == 0 && opts.Window != "" {
		windows, err := wm.reader.ListWindows(platform.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list windows: %w", err)
		}
		for _, w := range windows {
			if strings.Contains(strings.ToLower(w.Title), strings.ToLower(opts.Window)) {
				pid = w.PID
				break
			}
		}
		if pid == 0 {
			return fmt.Errorf("no window found matching title %q", opts.Window)
		}
	}

	if pid == 0 {
		return fmt.Errorf("could not resolve target: specify --app, --pid, --window, or --window-id")
	}

	// If a specific window is targeted, raise it via AX API
	if opts.Window != "" || opts.WindowID > 0 {
		var cTitle *C.char
		if opts.Window != "" {
			cTitle = C.CString(opts.Window)
			defer C.free(unsafe.Pointer(cTitle))
		}
		if C.ax_raise_window(C.pid_t(pid), cTitle, C.int(opts.WindowID)) != 0 {
			return fmt.Errorf("failed to raise window for PID %d", pid)
		}
		return nil
	}

	// Otherwise just activate the app
	if C.ns_activate_app(C.pid_t(pid)) != 0 {
		return fmt.Errorf("failed to activate app with PID %d", pid)
	}
	return nil
}

func (wm *DarwinWindowManager) GetFrontmostApp() (string, int, error) {
	var cName *C.char
	var cPid C.pid_t

	if C.ns_get_frontmost_app(&cName, &cPid) != 0 {
		return "", 0, fmt.Errorf("failed to get frontmost app")
	}
	defer C.free(unsafe.Pointer(cName))

	return C.GoString(cName), int(cPid), nil
}
