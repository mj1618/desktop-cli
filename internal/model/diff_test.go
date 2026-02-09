package model

import "testing"

func TestDiffElements_NoChanges(t *testing.T) {
	elements := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Bounds: [4]int{10, 20, 100, 30}, Path: "window"},
	}
	changes := DiffElements(elements, elements)
	if len(changes) != 0 {
		t.Errorf("expected no changes, got %d", len(changes))
	}
}

func TestDiffElements_Added(t *testing.T) {
	prev := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Path: "window"},
	}
	curr := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Path: "window"},
		{ID: 2, Role: "btn", Title: "Cancel", Path: "window"},
	}
	changes := DiffElements(prev, curr)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Type != ChangeAdded {
		t.Errorf("expected added, got %s", changes[0].Type)
	}
	if changes[0].Element.Title != "Cancel" {
		t.Errorf("expected Cancel, got %s", changes[0].Element.Title)
	}
}

func TestDiffElements_Removed(t *testing.T) {
	prev := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Path: "window"},
		{ID: 2, Role: "btn", Title: "Loading...", Path: "window"},
	}
	curr := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Path: "window"},
	}
	changes := DiffElements(prev, curr)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Type != ChangeRemoved {
		t.Errorf("expected removed, got %s", changes[0].Type)
	}
	if changes[0].ID != 2 {
		t.Errorf("expected ID 2, got %d", changes[0].ID)
	}
}

func TestDiffElements_Changed(t *testing.T) {
	prev := []FlatElement{
		{ID: 1, Role: "input", Title: "Search", Value: "", Path: "window"},
	}
	curr := []FlatElement{
		{ID: 1, Role: "input", Title: "Search", Value: "hello", Path: "window"},
	}
	changes := DiffElements(prev, curr)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Type != ChangeChanged {
		t.Errorf("expected changed, got %s", changes[0].Type)
	}
	if changes[0].Changes["v"][1] != "hello" {
		t.Errorf("expected new value 'hello', got %s", changes[0].Changes["v"][1])
	}
}

func TestDiffProperties_MultipleDiffs(t *testing.T) {
	prev := FlatElement{ID: 1, Role: "input", Title: "Name", Value: "old", Focused: false}
	curr := FlatElement{ID: 1, Role: "input", Title: "Name", Value: "new", Focused: true}
	diffs := diffProperties(prev, curr)
	if len(diffs) != 2 {
		t.Errorf("expected 2 diffs (v, f), got %d", len(diffs))
	}
	if diffs["v"][0] != "old" || diffs["v"][1] != "new" {
		t.Errorf("unexpected value diff: %v", diffs["v"])
	}
}

func TestDiffElements_Empty(t *testing.T) {
	changes := DiffElements(nil, nil)
	if len(changes) != 0 {
		t.Errorf("expected no changes for nil inputs, got %d", len(changes))
	}
}

func TestDiffElements_AllNew(t *testing.T) {
	curr := []FlatElement{
		{ID: 1, Role: "btn", Title: "A", Path: "window"},
		{ID: 2, Role: "txt", Title: "B", Path: "window"},
	}
	changes := DiffElements(nil, curr)
	if len(changes) != 2 {
		t.Fatalf("expected 2 added changes, got %d", len(changes))
	}
	for _, c := range changes {
		if c.Type != ChangeAdded {
			t.Errorf("expected added, got %s", c.Type)
		}
	}
}

func TestDiffElements_AllRemoved(t *testing.T) {
	prev := []FlatElement{
		{ID: 1, Role: "btn", Title: "A", Path: "window"},
		{ID: 2, Role: "txt", Title: "B", Path: "window"},
	}
	changes := DiffElements(prev, nil)
	if len(changes) != 2 {
		t.Fatalf("expected 2 removed changes, got %d", len(changes))
	}
	for _, c := range changes {
		if c.Type != ChangeRemoved {
			t.Errorf("expected removed, got %s", c.Type)
		}
	}
}

func TestDiffProperties_BoundsChange(t *testing.T) {
	prev := FlatElement{ID: 1, Role: "btn", Title: "OK", Bounds: [4]int{10, 20, 100, 30}}
	curr := FlatElement{ID: 1, Role: "btn", Title: "OK", Bounds: [4]int{10, 20, 200, 30}}
	diffs := diffProperties(prev, curr)
	if diffs == nil || diffs["b"][0] == "" {
		t.Error("expected bounds diff")
	}
}

func TestDiffProperties_SelectedChange(t *testing.T) {
	prev := FlatElement{ID: 1, Role: "row", Selected: false}
	curr := FlatElement{ID: 1, Role: "row", Selected: true}
	diffs := diffProperties(prev, curr)
	if diffs == nil || diffs["s"][1] != "true" {
		t.Error("expected selected diff")
	}
}

func TestDiffProperties_DescriptionChange(t *testing.T) {
	prev := FlatElement{ID: 1, Role: "img", Description: "old alt"}
	curr := FlatElement{ID: 1, Role: "img", Description: "new alt"}
	diffs := diffProperties(prev, curr)
	if diffs == nil || diffs["d"][0] != "old alt" || diffs["d"][1] != "new alt" {
		t.Errorf("expected description diff, got %v", diffs)
	}
}

func TestDiffProperties_NoDiff(t *testing.T) {
	el := FlatElement{ID: 1, Role: "btn", Title: "OK", Value: "v", Bounds: [4]int{1, 2, 3, 4}}
	diffs := diffProperties(el, el)
	if diffs != nil {
		t.Errorf("expected nil for identical elements, got %v", diffs)
	}
}
