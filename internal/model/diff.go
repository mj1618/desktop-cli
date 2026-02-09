package model

import (
	"fmt"
	"time"
)

// ChangeType represents the kind of UI change detected.
type ChangeType string

const (
	ChangeAdded   ChangeType = "added"
	ChangeRemoved ChangeType = "removed"
	ChangeChanged ChangeType = "changed"
)

// UIChange represents a single change between two reads.
type UIChange struct {
	Type    ChangeType           `json:"type"`
	TS      int64                `json:"ts"`
	Element *FlatElement         `json:"el,omitempty"`     // For added: the full element
	Path    string               `json:"p,omitempty"`      // For added: path in tree
	ID      int                  `json:"id,omitempty"`     // For removed/changed: element ID
	Role    string               `json:"r,omitempty"`      // For removed: role
	Title   string               `json:"t,omitempty"`      // For removed: title
	Changes map[string][2]string `json:"changes,omitempty"` // For changed: field diffs
}

// DiffElements compares two flat element lists and returns the changes.
// Elements are matched by their ID (sequential traversal index).
func DiffElements(prev, curr []FlatElement) []UIChange {
	prevMap := make(map[int]FlatElement, len(prev))
	for _, el := range prev {
		prevMap[el.ID] = el
	}
	currMap := make(map[int]FlatElement, len(curr))
	for _, el := range curr {
		currMap[el.ID] = el
	}

	var changes []UIChange
	now := time.Now().Unix()

	// Check for added and changed elements
	for _, el := range curr {
		prevEl, existed := prevMap[el.ID]
		if !existed {
			elCopy := el
			changes = append(changes, UIChange{
				Type:    ChangeAdded,
				TS:      now,
				Element: &elCopy,
				Path:    el.Path,
			})
			continue
		}
		diffs := diffProperties(prevEl, el)
		if len(diffs) > 0 {
			changes = append(changes, UIChange{
				Type:    ChangeChanged,
				TS:      now,
				ID:      el.ID,
				Changes: diffs,
			})
		}
	}

	// Check for removed elements
	for _, el := range prev {
		if _, exists := currMap[el.ID]; !exists {
			changes = append(changes, UIChange{
				Type:  ChangeRemoved,
				TS:    now,
				ID:    el.ID,
				Role:  el.Role,
				Title: el.Title,
			})
		}
	}

	return changes
}

// diffProperties compares two elements and returns changed fields.
func diffProperties(prev, curr FlatElement) map[string][2]string {
	diffs := make(map[string][2]string)

	if prev.Title != curr.Title {
		diffs["t"] = [2]string{prev.Title, curr.Title}
	}
	if prev.Value != curr.Value {
		diffs["v"] = [2]string{prev.Value, curr.Value}
	}
	if prev.Role != curr.Role {
		diffs["r"] = [2]string{prev.Role, curr.Role}
	}
	if prev.Description != curr.Description {
		diffs["d"] = [2]string{prev.Description, curr.Description}
	}
	if prev.Bounds != curr.Bounds {
		diffs["b"] = [2]string{
			fmt.Sprintf("%v", prev.Bounds),
			fmt.Sprintf("%v", curr.Bounds),
		}
	}
	if prev.Focused != curr.Focused {
		diffs["f"] = [2]string{
			fmt.Sprintf("%v", prev.Focused),
			fmt.Sprintf("%v", curr.Focused),
		}
	}
	if prev.Selected != curr.Selected {
		diffs["s"] = [2]string{
			fmt.Sprintf("%v", prev.Selected),
			fmt.Sprintf("%v", curr.Selected),
		}
	}

	if len(diffs) == 0 {
		return nil
	}
	return diffs
}
