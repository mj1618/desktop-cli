package model

import "strings"

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
		// Recursively filter children first
		var filteredChildren []Element
		if len(el.Children) > 0 {
			filteredChildren = FilterElements(el.Children, roles, bbox)
		}

		roleMatch := len(roleSet) == 0 || roleSet[el.Role]
		bboxMatch := bbox == nil || boundsIntersect(el.Bounds, *bbox)

		if roleMatch && bboxMatch {
			// Element matches filters: include it with filtered children
			filtered := el
			filtered.Children = filteredChildren
			result = append(result, filtered)
		} else if len(filteredChildren) > 0 {
			// Element doesn't match, but has matching descendants: include them directly
			result = append(result, filteredChildren...)
		}
	}
	return result
}

// FilterByText filters elements to only those whose title, value, or
// description contains the given text (case-insensitive). It recursively
// searches children and returns matching elements with their matching
// children preserved. Parent elements are included if any descendant matches.
func FilterByText(elements []Element, text string) []Element {
	if text == "" {
		return elements
	}
	textLower := strings.ToLower(text)
	var result []Element
	for _, el := range elements {
		matched := textMatchesElement(el, textLower)
		childMatches := FilterByText(el.Children, text)

		if matched || len(childMatches) > 0 {
			filtered := el
			filtered.Children = childMatches
			result = append(result, filtered)
		}
	}
	return result
}

func textMatchesElement(el Element, textLower string) bool {
	return strings.Contains(strings.ToLower(el.Title), textLower) ||
		strings.Contains(strings.ToLower(el.Value), textLower) ||
		strings.Contains(strings.ToLower(el.Description), textLower)
}

// FilterByFocused filters elements to only those that have Focused == true.
// It recursively searches children and returns matching elements with their
// ancestry path preserved (in tree mode) or just the focused element (in flat mode).
func FilterByFocused(elements []Element) []Element {
	var result []Element
	for _, el := range elements {
		childMatches := FilterByFocused(el.Children)

		if el.Focused {
			filtered := el
			filtered.Children = childMatches
			result = append(result, filtered)
		} else if len(childMatches) > 0 {
			filtered := el
			filtered.Children = childMatches
			result = append(result, filtered)
		}
	}
	return result
}

// isEmptyGroup returns true if the element has role "group" or "other"
// and has no title, value, or description — i.e. it carries no useful
// information for an agent.
func isEmptyGroup(el Element) bool {
	return (el.Role == "group" || el.Role == "other") &&
		el.Title == "" && el.Value == "" && el.Description == ""
}

// PruneEmptyGroups removes elements from a tree that are anonymous group/other
// nodes (no title, value, or description). Children of removed nodes are
// promoted to the parent. This dramatically reduces token usage when the
// accessibility tree contains many structural-only container nodes.
func PruneEmptyGroups(elements []Element) []Element {
	var result []Element
	for _, el := range elements {
		// Recursively prune children first
		prunedChildren := PruneEmptyGroups(el.Children)

		if isEmptyGroup(el) {
			// Skip this element, promote its children
			result = append(result, prunedChildren...)
		} else {
			pruned := el
			pruned.Children = prunedChildren
			result = append(result, pruned)
		}
	}
	return result
}

// PruneEmptyGroupsFlat removes FlatElements that are anonymous group/other
// nodes (no title, value, or description). The path breadcrumbs of remaining
// elements are not modified, preserving full ancestry context.
func PruneEmptyGroupsFlat(elements []FlatElement) []FlatElement {
	var result []FlatElement
	for _, el := range elements {
		if (el.Role == "group" || el.Role == "other") &&
			el.Title == "" && el.Value == "" && el.Description == "" {
			continue
		}
		result = append(result, el)
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
