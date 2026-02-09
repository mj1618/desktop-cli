//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation -framework Foundation
#include "action.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/mj1618/desktop-cli/internal/platform"
)

// DarwinActionPerformer implements the platform.ActionPerformer interface for macOS.
type DarwinActionPerformer struct {
	reader *DarwinReader
}

// NewActionPerformer creates a new macOS action performer.
func NewActionPerformer(reader *DarwinReader) *DarwinActionPerformer {
	return &DarwinActionPerformer{reader: reader}
}

func (p *DarwinActionPerformer) PerformAction(opts platform.ActionOptions) error {
	if opts.ID <= 0 {
		return fmt.Errorf("--id is required")
	}
	if opts.Action == "" {
		return fmt.Errorf("--action is required")
	}

	if err := CheckAccessibilityPermission(); err != nil {
		return err
	}

	// Resolve PID and window using the same logic as ReadElements
	readOpts := platform.ReadOptions{
		App:      opts.App,
		Window:   opts.Window,
		WindowID: opts.WindowID,
		PID:      opts.PID,
	}
	pid, windowTitle, windowID := p.reader.resolvePIDAndWindow(readOpts)
	if pid == 0 {
		return fmt.Errorf("no target specified: use --app, --pid, --window, or --window-id")
	}

	// Map short action name to full AX action name
	axAction := mapActionName(opts.Action)

	cAction := C.CString(axAction)
	defer C.free(unsafe.Pointer(cAction))

	cWindowTitle := (*C.char)(nil)
	if windowTitle != "" {
		cWindowTitle = C.CString(windowTitle)
		defer C.free(unsafe.Pointer(cWindowTitle))
	}

	rc := C.ax_perform_action(C.pid_t(pid), cWindowTitle, C.int(windowID),
		C.int(0), C.int(opts.ID), cAction)
	if rc != 0 {
		return fmt.Errorf("failed to perform action %q on element %d", opts.Action, opts.ID)
	}

	return nil
}

func mapActionName(short string) string {
	switch strings.ToLower(short) {
	case "press":
		return "AXPress"
	case "cancel":
		return "AXCancel"
	case "pick":
		return "AXPick"
	case "increment":
		return "AXIncrement"
	case "decrement":
		return "AXDecrement"
	case "confirm":
		return "AXConfirm"
	case "showmenu":
		return "AXShowMenu"
	case "raise":
		return "AXRaise"
	default:
		return short
	}
}
