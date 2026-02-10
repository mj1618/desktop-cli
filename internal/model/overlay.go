package model

// overlaySubroles are macOS AXSubrole values that indicate an overlay/dialog/sheet.
var overlaySubroles = map[string]bool{
	"AXDialog":         true,
	"AXSheet":          true,
	"AXSystemDialog":   true,
	"AXSystemFloating": true,
	"AXFloatingWindow": true,
}

// DetectFrontmostOverlay examines the top-level children of the element tree
// to find a modal dialog, sheet, or popover that should be the auto-scope target.
//
// Detection strategies (tried in order):
//  1. Subrole-based: Any top-level child with an overlay subrole (AXDialog, AXSheet, etc.)
//  2. Focus-based: Find the focused element, walk up to its nearest window-level ancestor.
//     If that ancestor is NOT the first (main) child, it's likely an overlay.
//  3. Bounds-based: A top-level element that is centered within and smaller than the window
//     (typical dialog pattern).
//
// Returns the overlay element, or nil if no overlay is detected.
// Only examines children of window elements (not the windows themselves).
func DetectFrontmostOverlay(elements []Element) *Element {
	// Work with window children — each top-level element is typically a window
	for i := range elements {
		win := &elements[i]
		if win.Role != "window" {
			continue
		}
		if overlay := detectOverlayInWindow(win); overlay != nil {
			return overlay
		}
	}
	return nil
}

// detectOverlayInWindow checks a single window's children for overlays.
func detectOverlayInWindow(win *Element) *Element {
	if len(win.Children) == 0 {
		return nil
	}

	// Strategy 1: Look for children with overlay subroles
	for i := range win.Children {
		child := &win.Children[i]
		if overlaySubroles[child.Subrole] {
			return child
		}
		// Also check grandchildren — some apps nest the dialog one level deeper
		for j := range child.Children {
			gc := &child.Children[j]
			if overlaySubroles[gc.Subrole] {
				return gc
			}
		}
	}

	// Strategy 2: Focus-based — find the focused element and check if its
	// top-level ancestor (direct child of the window) differs from the main content.
	// If the window has multiple direct children and focus is in a non-first child,
	// that child is likely an overlay.
	if len(win.Children) > 1 {
		focusedChildIdx := findFocusedChildIndex(win.Children)
		if focusedChildIdx > 0 {
			candidate := &win.Children[focusedChildIdx]
			// Verify it looks like an overlay: smaller than the window and centered-ish
			if isOverlaySized(candidate, win) {
				return candidate
			}
		}
	}

	// Strategy 3: Bounds-based — look for a child that appears to be a centered dialog
	// (smaller than the window and positioned near its center).
	if len(win.Children) > 1 {
		for i := 1; i < len(win.Children); i++ {
			child := &win.Children[i]
			if isOverlaySized(child, win) && isCentered(child, win) {
				return child
			}
		}
	}

	return nil
}

// findFocusedChildIndex returns the index of the direct child that contains
// the focused element. Returns -1 if no focused element is found.
func findFocusedChildIndex(children []Element) int {
	for i := range children {
		if containsFocused(&children[i]) {
			return i
		}
	}
	return -1
}

// containsFocused recursively checks if an element or any descendant has focus.
func containsFocused(el *Element) bool {
	if el.Focused {
		return true
	}
	for i := range el.Children {
		if containsFocused(&el.Children[i]) {
			return true
		}
	}
	return false
}

// isOverlaySized returns true if the candidate element is meaningfully smaller
// than the window (at least 20% smaller in both dimensions), suggesting it's
// a dialog/overlay rather than the main content area.
func isOverlaySized(candidate, window *Element) bool {
	winW, winH := window.Bounds[2], window.Bounds[3]
	candW, candH := candidate.Bounds[2], candidate.Bounds[3]

	if winW == 0 || winH == 0 || candW == 0 || candH == 0 {
		return false
	}

	// Candidate must be smaller than 80% of the window in at least one dimension
	return candW < winW*80/100 || candH < winH*80/100
}

// isCentered returns true if the candidate element is roughly centered within
// the window (its center is within 25% of the window's center).
func isCentered(candidate, window *Element) bool {
	winCX := window.Bounds[0] + window.Bounds[2]/2
	winCY := window.Bounds[1] + window.Bounds[3]/2
	candCX := candidate.Bounds[0] + candidate.Bounds[2]/2
	candCY := candidate.Bounds[1] + candidate.Bounds[3]/2

	threshX := window.Bounds[2] / 4
	threshY := window.Bounds[3] / 4

	dx := candCX - winCX
	dy := candCY - winCY
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}

	return dx <= threshX && dy <= threshY
}
