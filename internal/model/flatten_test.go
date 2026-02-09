package model

import "testing"

func TestFlattenElements_Basic(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Title: "OK", Bounds: [4]int{0, 0, 100, 30}},
		{ID: 2, Role: "txt", Title: "Hello", Bounds: [4]int{0, 30, 100, 20}},
	}
	result := FlattenElements(elements)
	if len(result) != 2 {
		t.Fatalf("expected 2 flat elements, got %d", len(result))
	}
	if result[0].Path != "btn" {
		t.Errorf("expected path 'btn', got %q", result[0].Path)
	}
	if result[1].Path != "txt" {
		t.Errorf("expected path 'txt', got %q", result[1].Path)
	}
}

func TestFlattenElements_NestedPath(t *testing.T) {
	elements := []Element{
		{
			ID: 1, Role: "window", Title: "Main",
			Children: []Element{
				{
					ID: 2, Role: "toolbar", Title: "Nav",
					Children: []Element{
						{ID: 3, Role: "btn", Title: "Back"},
					},
				},
			},
		},
	}
	result := FlattenElements(elements)
	if len(result) != 3 {
		t.Fatalf("expected 3 flat elements, got %d", len(result))
	}
	if result[0].Path != "window" {
		t.Errorf("expected path 'window', got %q", result[0].Path)
	}
	if result[1].Path != "window > toolbar" {
		t.Errorf("expected path 'window > toolbar', got %q", result[1].Path)
	}
	if result[2].Path != "window > toolbar > btn" {
		t.Errorf("expected path 'window > toolbar > btn', got %q", result[2].Path)
	}
}

func TestFlattenElements_PreservesIDs(t *testing.T) {
	elements := []Element{
		{
			ID: 1, Role: "group",
			Children: []Element{
				{ID: 5, Role: "btn", Title: "Submit"},
				{ID: 10, Role: "txt", Title: "Label"},
			},
		},
	}
	result := FlattenElements(elements)
	if len(result) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(result))
	}
	if result[0].ID != 1 {
		t.Errorf("expected ID 1, got %d", result[0].ID)
	}
	if result[1].ID != 5 {
		t.Errorf("expected ID 5, got %d", result[1].ID)
	}
	if result[2].ID != 10 {
		t.Errorf("expected ID 10, got %d", result[2].ID)
	}
}

func TestFlattenElements_NoChildren(t *testing.T) {
	result := FlattenElements(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 elements for nil input, got %d", len(result))
	}
}

func TestFlattenElements_PreservesFields(t *testing.T) {
	f := false
	elements := []Element{
		{
			ID:          1,
			Role:        "input",
			Title:       "Search",
			Value:       "hello",
			Description: "Search field",
			Bounds:      [4]int{100, 200, 300, 40},
			Focused:     true,
			Enabled:     &f,
			Selected:    true,
			Actions:     []string{"press"},
		},
	}
	result := FlattenElements(elements)
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	el := result[0]
	if el.Title != "Search" {
		t.Errorf("expected title 'Search', got %q", el.Title)
	}
	if el.Value != "hello" {
		t.Errorf("expected value 'hello', got %q", el.Value)
	}
	if el.Description != "Search field" {
		t.Errorf("expected description 'Search field', got %q", el.Description)
	}
	if el.Bounds != [4]int{100, 200, 300, 40} {
		t.Errorf("unexpected bounds: %v", el.Bounds)
	}
	if !el.Focused {
		t.Error("expected focused=true")
	}
	if el.Enabled == nil || *el.Enabled != false {
		t.Error("expected enabled=false")
	}
	if !el.Selected {
		t.Error("expected selected=true")
	}
	if len(el.Actions) != 1 || el.Actions[0] != "press" {
		t.Errorf("unexpected actions: %v", el.Actions)
	}
	if el.Path != "input" {
		t.Errorf("expected path 'input', got %q", el.Path)
	}
}

func TestFlattenElements_TraversalOrder(t *testing.T) {
	elements := []Element{
		{
			ID: 1, Role: "window",
			Children: []Element{
				{
					ID: 2, Role: "group",
					Children: []Element{
						{ID: 3, Role: "btn", Title: "A"},
					},
				},
				{ID: 4, Role: "btn", Title: "B"},
			},
		},
	}
	result := FlattenElements(elements)
	if len(result) != 4 {
		t.Fatalf("expected 4 elements, got %d", len(result))
	}
	expectedIDs := []int{1, 2, 3, 4}
	for i, want := range expectedIDs {
		if result[i].ID != want {
			t.Errorf("element %d: expected ID %d, got %d", i, want, result[i].ID)
		}
	}
}
