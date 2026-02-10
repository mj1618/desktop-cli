package model

import (
	"fmt"
	"regexp"
	"strings"
)

// slugRe matches characters that are not lowercase alphanumeric or hyphens.
var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

// slugify converts a label to a URL-safe slug: lowercase, hyphens for spaces/special chars.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	// Collapse multiple hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	// Truncate long slugs
	if len(s) > 40 {
		s = s[:40]
		s = strings.TrimRight(s, "-")
	}
	return s
}

// bestLabel returns the best stable label for an element: title > description.
// Value is excluded because it changes (input field content, slider position).
func bestLabel(el Element) string {
	if el.Title != "" {
		return el.Title
	}
	if el.Description != "" {
		return el.Description
	}
	return ""
}

// landmarkRoles are roles that are always kept in the ref path as landmarks.
var landmarkRoles = map[string]bool{
	"toolbar": true,
	"menu":    true,
	"list":    true,
	"tab":     true,
}

// dialogSubroles are subroles that indicate a dialog/overlay landmark.
var dialogSubroles = map[string]bool{
	"AXDialog":       true,
	"AXSheet":        true,
	"AXSystemDialog": true,
}

// skippedRoles are roles that are always skipped in the ref path.
var skippedRoles = map[string]bool{
	"window": true,
	"scroll": true,
	"web":    true,
}

// isLandmark returns true if the element should be kept as a landmark in the ref path.
func isLandmark(el Element) bool {
	if landmarkRoles[el.Role] {
		return true
	}
	if dialogSubroles[el.Subrole] {
		return true
	}
	// Labeled groups are landmarks
	if el.Role == "group" && bestLabel(el) != "" {
		return true
	}
	return false
}

// isSkippedInPath returns true if the element should be skipped (not contribute to path).
func isSkippedInPath(el Element) bool {
	if skippedRoles[el.Role] {
		return true
	}
	// Unlabeled structural roles are skipped
	if bestLabel(el) == "" {
		switch el.Role {
		case "group", "other", "row", "cell":
			// Skip unlabeled structural elements unless they're landmarks
			return !isLandmark(el)
		}
	}
	return false
}

// refSegment returns the path segment for an element.
func refSegment(el Element) string {
	label := bestLabel(el)
	if label != "" {
		slug := slugify(label)
		if slug != "" {
			return slug
		}
	}
	// Fall back to role for elements with no useful label
	if dialogSubroles[el.Subrole] {
		return "dialog"
	}
	return el.Role
}

// GenerateRefs walks the element tree and populates the Ref field on each
// element. Refs are stable path-based identifiers like "toolbar/search" or
// "dialog/submit" that persist across reads as long as the element's semantic
// identity doesn't change.
//
// Only "interesting" elements get refs (interactive elements with "press" action,
// or display text elements). Container/structural elements don't get refs but
// contribute to the path.
func GenerateRefs(elements []Element) {
	// First pass: generate raw refs
	generateRefsRecursive(elements, "")

	// Second pass: deduplicate
	deduplicateRefs(elements)
}

func generateRefsRecursive(elements []Element, parentPath string) {
	for i := range elements {
		el := &elements[i]

		// Determine what this element contributes to the path for its children
		var childPath string
		if isLandmark(*el) {
			seg := refSegment(*el)
			if parentPath != "" {
				childPath = parentPath + "/" + seg
			} else {
				childPath = seg
			}
		} else if isSkippedInPath(*el) {
			childPath = parentPath
		} else {
			// Non-landmark, non-skipped element with a label (e.g. labeled btn, input)
			// It doesn't extend the path for children, just uses the current path
			childPath = parentPath
		}

		// Assign ref to interesting elements (interactive or display text)
		if isInteresting(*el) {
			seg := refSegment(*el)
			if parentPath != "" {
				el.Ref = parentPath + "/" + seg
			} else {
				el.Ref = seg
			}
		}

		// Recurse into children
		generateRefsRecursive(el.Children, childPath)
	}
}

// isInteresting returns true if an element should get a ref.
// Same criteria as the agent format: interactive (has "press" action) or display text.
func isInteresting(el Element) bool {
	for _, a := range el.Actions {
		if a == "press" {
			return true
		}
	}
	// Display text: txt role with a value
	if el.Role == "txt" && el.Value != "" {
		return true
	}
	// Input fields, checkboxes, toggles, radios, sliders are interesting
	switch el.Role {
	case "input", "chk", "toggle", "radio":
		return true
	}
	return false
}

// deduplicateRefs finds elements with identical refs and appends .1, .2 suffixes.
func deduplicateRefs(elements []Element) {
	// Collect all refs with their element pointers
	refCounts := make(map[string][]*Element)
	collectRefs(elements, refCounts)

	// For any ref that appears more than once, append index suffixes
	for ref, elems := range refCounts {
		if len(elems) <= 1 {
			continue
		}
		for i, el := range elems {
			el.Ref = fmt.Sprintf("%s.%d", ref, i+1)
		}
	}
}

func collectRefs(elements []Element, refCounts map[string][]*Element) {
	for i := range elements {
		if elements[i].Ref != "" {
			refCounts[elements[i].Ref] = append(refCounts[elements[i].Ref], &elements[i])
		}
		collectRefs(elements[i].Children, refCounts)
	}
}

// refEntry pairs a ref string with its element pointer.
type refEntry struct {
	ref string
	el  *Element
}

// FindElementByRef searches a ref-populated element tree for the element matching
// the given ref. Supports exact match and partial suffix match.
// Returns the matched element or nil if not found / ambiguous.
func FindElementByRef(elements []Element, ref string) (*Element, error) {
	var entries []refEntry
	collectRefEntries(elements, &entries)

	// Exact match first
	for _, e := range entries {
		if e.ref == ref {
			return e.el, nil
		}
	}

	// Partial match: find refs ending with the provided value
	var matches []refEntry
	for _, e := range entries {
		if strings.HasSuffix(e.ref, "/"+ref) {
			matches = append(matches, e)
		}
	}

	if len(matches) == 1 {
		return matches[0].el, nil
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no element matches ref %q", ref)
	}

	// Multiple matches — build helpful error
	var b strings.Builder
	fmt.Fprintf(&b, "multiple elements match ref %q:\n", ref)
	for _, m := range matches {
		fmt.Fprintf(&b, "  ref=%q id=%d %s", m.ref, m.el.ID, m.el.Role)
		if m.el.Title != "" {
			fmt.Fprintf(&b, " title=%q", m.el.Title)
		}
		fmt.Fprintln(&b)
	}
	return nil, fmt.Errorf("%s", b.String())
}

func collectRefEntries(elements []Element, entries *[]refEntry) {
	for i := range elements {
		if elements[i].Ref != "" {
			*entries = append(*entries, refEntry{ref: elements[i].Ref, el: &elements[i]})
		}
		collectRefEntries(elements[i].Children, entries)
	}
}
