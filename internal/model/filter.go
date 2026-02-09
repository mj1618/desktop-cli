package model

// FilterElements applies filters to a slice of elements, returning only
// matching elements. It filters by roles and bounding box. Depth filtering
// should happen during traversal, not here.
//
// To avoid circular imports between model and platform, this function accepts
// individual filter parameters rather than the full ReadOptions struct.
func FilterElements(elements []Element, roles []string, bbox *[4]int) []Element {
	if len(roles) == 0 && bbox == nil {
		return elements
	}

	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	var result []Element
	for _, el := range elements {
		if len(roleSet) > 0 && !roleSet[el.Role] {
			continue
		}
		if bbox != nil && !boundsIntersect(el.Bounds, *bbox) {
			continue
		}
		// Recursively filter children
		filtered := el
		if len(el.Children) > 0 {
			filtered.Children = FilterElements(el.Children, roles, bbox)
		}
		result = append(result, filtered)
	}
	return result
}

// boundsIntersect checks if two [x, y, width, height] rectangles overlap.
func boundsIntersect(a, b [4]int) bool {
	// a and b are [x, y, width, height]
	ax1, ay1, ax2, ay2 := a[0], a[1], a[0]+a[2], a[1]+a[3]
	bx1, by1, bx2, by2 := b[0], b[1], b[0]+b[2], b[1]+b[3]
	return ax1 < bx2 && ax2 > bx1 && ay1 < by2 && ay2 > by1
}
