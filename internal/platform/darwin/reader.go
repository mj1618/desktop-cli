//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ApplicationServices -framework Foundation
#include "window_list.h"
#include "accessibility.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"sort"
	"strings"
	"unsafe"

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/platform"
)

// DarwinReader implements the platform.Reader interface for macOS.
type DarwinReader struct{}

// NewReader creates a new macOS reader.
func NewReader() *DarwinReader {
	return &DarwinReader{}
}

// ListWindows returns all windows visible on screen using CGWindowListCopyWindowInfo.
// Filters by app name and PID per ListOptions.
func (r *DarwinReader) ListWindows(opts platform.ListOptions) ([]model.Window, error) {
	var cWindows *C.CGWindowInfo
	var cCount C.int

	if C.cg_list_windows(&cWindows, &cCount) != 0 {
		return nil, fmt.Errorf("failed to enumerate windows")
	}
	defer C.cg_free_windows(cWindows, cCount)

	count := int(cCount)
	if count == 0 {
		return []model.Window{}, nil
	}

	// Get the frontmost app PID to determine focused state
	frontPid := int(C.cg_get_frontmost_pid())

	// Convert C array to Go slice for access
	cSlice := unsafe.Slice(cWindows, count)

	// Track if we've already marked a window as focused for the frontmost app
	frontmostFocusAssigned := false

	var windows []model.Window
	for i := 0; i < count; i++ {
		cw := cSlice[i]

		// Filter to layer 0 only (real application windows)
		if int(cw.layer) != 0 {
			continue
		}

		appName := C.GoString(cw.appName)
		title := C.GoString(cw.title)
		pid := int(cw.pid)
		windowID := int(cw.windowID)

		// Apply filters
		if opts.PID != 0 && pid != opts.PID {
			continue
		}
		if opts.App != "" && !strings.EqualFold(appName, opts.App) {
			continue
		}

		// Determine focused state: first window of the frontmost app is focused
		focused := false
		if pid == frontPid && !frontmostFocusAssigned {
			focused = true
			frontmostFocusAssigned = true
		}

		w := model.Window{
			App:   appName,
			PID:   pid,
			Title: title,
			ID:    windowID,
			Bounds: [4]int{
				int(cw.x),
				int(cw.y),
				int(cw.width),
				int(cw.height),
			},
			Focused: focused,
		}
		windows = append(windows, w)
	}

	if windows == nil {
		windows = []model.Window{}
	}

	// Enrich empty titles using the Accessibility API.
	// Group windows with empty titles by PID so we query each app only once.
	pidsNeedingTitles := make(map[int]bool)
	for _, w := range windows {
		if w.Title == "" {
			pidsNeedingTitles[w.PID] = true
		}
	}
	if len(pidsNeedingTitles) > 0 {
		// Build a windowID->title map from accessibility API
		axTitleMap := make(map[int]string)
		for pid := range pidsNeedingTitles {
			var axTitles *C.AXWindowTitle
			var axCount C.int
			if C.ax_list_window_titles(C.pid_t(pid), &axTitles, &axCount) == 0 && axCount > 0 {
				axSlice := unsafe.Slice(axTitles, int(axCount))
				for j := 0; j < int(axCount); j++ {
					t := C.GoString(axSlice[j].title)
					if t != "" {
						axTitleMap[int(axSlice[j].windowID)] = t
					}
				}
				C.ax_free_window_titles(axTitles, axCount)
			}
		}
		// Fill in empty titles
		for i := range windows {
			if windows[i].Title == "" {
				if t, ok := axTitleMap[windows[i].ID]; ok {
					windows[i].Title = t
				}
			}
		}
	}

	// Sort: focused first, then by app name
	sort.Slice(windows, func(i, j int) bool {
		if windows[i].Focused != windows[j].Focused {
			return windows[i].Focused
		}
		return strings.ToLower(windows[i].App) < strings.ToLower(windows[j].App)
	})

	return windows, nil
}

// ActionMap maps AX action names to short names.
var ActionMap = map[string]string{
	"AXPress":     "press",
	"AXCancel":    "cancel",
	"AXPick":      "pick",
	"AXIncrement": "increment",
	"AXDecrement": "decrement",
	"AXConfirm":   "confirm",
	"AXShowMenu":  "showmenu",
}

func mapAction(axAction string) string {
	if short, ok := ActionMap[axAction]; ok {
		return short
	}
	return strings.ToLower(strings.TrimPrefix(axAction, "AX"))
}

// ReadElements reads the accessibility element tree for the specified target.
func (r *DarwinReader) ReadElements(opts platform.ReadOptions) ([]model.Element, error) {
	if err := CheckAccessibilityPermission(); err != nil {
		return nil, err
	}

	pid, windowTitle, windowID := r.resolvePIDAndWindow(opts)
	if pid == 0 {
		return nil, fmt.Errorf("no target specified: use --app, --pid, --window, or --window-id")
	}

	var cElements *C.AXElementInfo
	var cCount C.int
	cWindowTitle := (*C.char)(nil)
	if windowTitle != "" {
		cWindowTitle = C.CString(windowTitle)
		defer C.free(unsafe.Pointer(cWindowTitle))
	}

	if C.ax_read_elements(C.pid_t(pid), cWindowTitle, C.int(windowID),
		C.int(opts.Depth), &cElements, &cCount) != 0 {
		return nil, fmt.Errorf("failed to read accessibility tree for PID %d", pid)
	}
	defer C.ax_free_elements(cElements, cCount)

	elements := buildElementTree(cElements, cCount)

	// Apply role and bbox filters
	var bbox *[4]int
	if opts.BBox != nil {
		b := [4]int{opts.BBox.X, opts.BBox.Y, opts.BBox.Width, opts.BBox.Height}
		bbox = &b
	}
	elements = model.FilterElements(elements, opts.Roles, bbox)

	return elements, nil
}

// resolvePIDAndWindow resolves the target PID from --app, --pid, --window, or --window-id.
func (r *DarwinReader) resolvePIDAndWindow(opts platform.ReadOptions) (pid int, windowTitle string, windowID int) {
	if opts.PID != 0 {
		return opts.PID, opts.Window, opts.WindowID
	}
	if opts.App != "" {
		windows, err := r.ListWindows(platform.ListOptions{App: opts.App})
		if err != nil || len(windows) == 0 {
			return 0, "", 0
		}
		if opts.Window != "" {
			for _, w := range windows {
				if strings.Contains(strings.ToLower(w.Title), strings.ToLower(opts.Window)) {
					return w.PID, "", w.ID
				}
			}
		}
		return windows[0].PID, opts.Window, opts.WindowID
	}
	if opts.WindowID != 0 {
		windows, err := r.ListWindows(platform.ListOptions{})
		if err != nil {
			return 0, "", 0
		}
		for _, w := range windows {
			if w.ID == opts.WindowID {
				return w.PID, "", w.ID
			}
		}
	}
	if opts.Window != "" {
		windows, err := r.ListWindows(platform.ListOptions{})
		if err != nil {
			return 0, "", 0
		}
		for _, w := range windows {
			if strings.Contains(strings.ToLower(w.Title), strings.ToLower(opts.Window)) {
				return w.PID, "", w.ID
			}
		}
	}
	return 0, "", 0
}

// buildElementTree converts the flat C array into a nested Go element tree.
func buildElementTree(cElements *C.AXElementInfo, cCount C.int) []model.Element {
	count := int(cCount)
	if count == 0 {
		return []model.Element{}
	}

	cSlice := unsafe.Slice(cElements, count)

	// Build flat list of elements
	type elemEntry struct {
		elem     model.Element
		parentID int
	}
	entries := make([]elemEntry, count)
	elemMap := make(map[int]int, count) // id -> index in entries
	var roots []int

	for i := 0; i < count; i++ {
		ce := cSlice[i]
		id := int(ce.id)
		parentID := int(ce.parentID)

		role := model.MapRole(C.GoString(ce.role))

		var actions []string
		if ce.actionCount > 0 {
			cActions := unsafe.Slice(ce.actions, int(ce.actionCount))
			for j := 0; j < int(ce.actionCount); j++ {
				actions = append(actions, mapAction(C.GoString(cActions[j])))
			}
		}

		var enabled *bool
		if ce.enabled == 0 {
			f := false
			enabled = &f
		}

		subrole := C.GoString(ce.subrole)

		entries[i] = elemEntry{
			elem: model.Element{
				ID:          id,
				Role:        role,
				Subrole:     subrole,
				Title:       C.GoString(ce.title),
				Value:       C.GoString(ce.value),
				Description: C.GoString(ce.description),
				Bounds:      [4]int{int(ce.x), int(ce.y), int(ce.width), int(ce.height)},
				Focused:     ce.focused != 0,
				Enabled:     enabled,
				Selected:    ce.selected != 0,
				Actions:     actions,
			},
			parentID: parentID,
		}
		elemMap[id] = i

		if parentID < 0 {
			roots = append(roots, i)
		}
	}

	// Build tree bottom-up: assign children to parents
	// Process in reverse order so children are added in order
	for i := count - 1; i >= 0; i-- {
		if entries[i].parentID >= 0 {
			if pIdx, ok := elemMap[entries[i].parentID]; ok {
				entries[pIdx].elem.Children = append([]model.Element{entries[i].elem}, entries[pIdx].elem.Children...)
			}
		}
	}

	// Update children recursively since we built with copies
	// We need to rebuild from roots using the final entries
	var buildTree func(idx int) model.Element
	buildTree = func(idx int) model.Element {
		elem := entries[idx].elem
		if len(elem.Children) > 0 {
			rebuilt := make([]model.Element, 0, len(elem.Children))
			for _, child := range elem.Children {
				if childIdx, ok := elemMap[child.ID]; ok {
					rebuilt = append(rebuilt, buildTree(childIdx))
				}
			}
			elem.Children = rebuilt
		}
		return elem
	}

	result := make([]model.Element, 0, len(roots))
	for _, ri := range roots {
		result = append(result, buildTree(ri))
	}
	return result
}
