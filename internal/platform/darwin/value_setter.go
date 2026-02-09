//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation -framework Foundation
#include "set_value.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/mj1618/desktop-cli/internal/platform"
)

// DarwinValueSetter implements the platform.ValueSetter interface for macOS.
type DarwinValueSetter struct {
	reader *DarwinReader
}

// NewValueSetter creates a new macOS value setter.
func NewValueSetter(reader *DarwinReader) *DarwinValueSetter {
	return &DarwinValueSetter{reader: reader}
}

func (s *DarwinValueSetter) SetValue(opts platform.SetValueOptions) error {
	if opts.ID <= 0 {
		return fmt.Errorf("--id is required")
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
	pid, windowTitle, windowID := s.reader.resolvePIDAndWindow(readOpts)
	if pid == 0 {
		return fmt.Errorf("no target specified: use --app, --pid, --window, or --window-id")
	}

	// Map short attribute name to full AX attribute name
	attribute := "AXValue"
	switch opts.Attribute {
	case "", "value":
		attribute = "AXValue"
	case "selected":
		attribute = "AXSelected"
	case "focused":
		attribute = "AXFocused"
	default:
		attribute = opts.Attribute
	}

	cWindowTitle := (*C.char)(nil)
	if windowTitle != "" {
		cWindowTitle = C.CString(windowTitle)
		defer C.free(unsafe.Pointer(cWindowTitle))
	}

	cAttribute := C.CString(attribute)
	defer C.free(unsafe.Pointer(cAttribute))

	cValue := C.CString(opts.Value)
	defer C.free(unsafe.Pointer(cValue))

	rc := C.ax_set_value(C.pid_t(pid), cWindowTitle, C.int(windowID),
		C.int(0), C.int(opts.ID), cAttribute, cValue)
	if rc != 0 {
		return fmt.Errorf("failed to set %s=%q on element %d", attribute, opts.Value, opts.ID)
	}

	return nil
}
