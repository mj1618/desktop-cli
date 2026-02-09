package model

import "testing"

func TestFilterElements_NoFilters(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Bounds: [4]int{0, 0, 100, 30}},
		{ID: 2, Role: "txt", Bounds: [4]int{0, 30, 100, 20}},
	}
	result := FilterElements(elements, nil, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 elements, got %d", len(result))
	}
}

func TestFilterElements_RoleFilter(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Bounds: [4]int{0, 0, 100, 30}},
		{ID: 2, Role: "txt", Bounds: [4]int{0, 30, 100, 20}},
		{ID: 3, Role: "lnk", Bounds: [4]int{0, 50, 100, 20}},
	}
	result := FilterElements(elements, []string{"btn", "lnk"}, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 elements, got %d", len(result))
	}
	if result[0].Role != "btn" || result[1].Role != "lnk" {
		t.Errorf("unexpected roles: %s, %s", result[0].Role, result[1].Role)
	}
}

func TestFilterElements_BBoxFilter(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Bounds: [4]int{10, 10, 50, 30}},   // inside
		{ID: 2, Role: "btn", Bounds: [4]int{200, 200, 50, 30}},  // outside
		{ID: 3, Role: "btn", Bounds: [4]int{90, 90, 50, 30}},    // overlaps
	}
	bbox := [4]int{0, 0, 100, 100}
	result := FilterElements(elements, nil, &bbox)
	if len(result) != 2 {
		t.Errorf("expected 2 elements (inside + overlapping), got %d", len(result))
	}
}

func TestFilterElements_RecursiveChildren(t *testing.T) {
	elements := []Element{
		{
			ID: 1, Role: "group", Bounds: [4]int{0, 0, 200, 200},
			Children: []Element{
				{ID: 2, Role: "btn", Bounds: [4]int{10, 10, 50, 30}},
				{ID: 3, Role: "txt", Bounds: [4]int{10, 50, 100, 20}},
			},
		},
	}
	result := FilterElements(elements, []string{"group", "btn"}, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 top-level element, got %d", len(result))
	}
	if len(result[0].Children) != 1 {
		t.Errorf("expected 1 child after filtering, got %d", len(result[0].Children))
	}
	if result[0].Children[0].Role != "btn" {
		t.Errorf("expected child role btn, got %s", result[0].Children[0].Role)
	}
}

func TestFilterElements_NestedRoleFilter(t *testing.T) {
	// Buttons nested inside groups/toolbars should be found even if
	// the parent roles are not in the filter set.
	elements := []Element{
		{
			ID: 1, Role: "window", Bounds: [4]int{0, 0, 800, 600},
			Children: []Element{
				{
					ID: 2, Role: "toolbar", Bounds: [4]int{0, 0, 800, 50},
					Children: []Element{
						{ID: 3, Role: "btn", Title: "Back", Bounds: [4]int{10, 10, 60, 30}},
						{ID: 4, Role: "btn", Title: "Forward", Bounds: [4]int{80, 10, 60, 30}},
						{ID: 5, Role: "txt", Title: "URL", Bounds: [4]int{150, 10, 400, 30}},
					},
				},
			},
		},
	}
	result := FilterElements(elements, []string{"btn"}, nil)
	if len(result) != 2 {
		t.Fatalf("expected 2 buttons found through nested parents, got %d", len(result))
	}
	if result[0].Title != "Back" || result[1].Title != "Forward" {
		t.Errorf("unexpected buttons: %s, %s", result[0].Title, result[1].Title)
	}
}

func TestFilterByText_EmptyText(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Title: "OK"},
		{ID: 2, Role: "txt", Title: "Hello"},
	}
	result := FilterByText(elements, "")
	if len(result) != 2 {
		t.Errorf("expected 2 elements for empty text, got %d", len(result))
	}
}

func TestFilterByText_TitleMatch(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Title: "Submit"},
		{ID: 2, Role: "btn", Title: "Cancel"},
		{ID: 3, Role: "txt", Title: "Submit Review"},
	}
	result := FilterByText(elements, "Submit")
	if len(result) != 2 {
		t.Errorf("expected 2 matching elements, got %d", len(result))
	}
	if result[0].ID != 1 || result[1].ID != 3 {
		t.Errorf("unexpected IDs: %d, %d", result[0].ID, result[1].ID)
	}
}

func TestFilterByText_CaseInsensitive(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Title: "SUBMIT"},
		{ID: 2, Role: "btn", Title: "submit"},
		{ID: 3, Role: "btn", Title: "Submit"},
	}
	result := FilterByText(elements, "submit")
	if len(result) != 3 {
		t.Errorf("expected 3 case-insensitive matches, got %d", len(result))
	}
}

func TestFilterByText_ValueMatch(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "input", Value: "hello world"},
		{ID: 2, Role: "input", Value: "goodbye"},
	}
	result := FilterByText(elements, "hello")
	if len(result) != 1 {
		t.Errorf("expected 1 match on value, got %d", len(result))
	}
	if result[0].ID != 1 {
		t.Errorf("expected ID 1, got %d", result[0].ID)
	}
}

func TestFilterByText_DescriptionMatch(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "img", Description: "Profile photo"},
		{ID: 2, Role: "img", Description: "Logo"},
	}
	result := FilterByText(elements, "profile")
	if len(result) != 1 {
		t.Errorf("expected 1 match on description, got %d", len(result))
	}
	if result[0].ID != 1 {
		t.Errorf("expected ID 1, got %d", result[0].ID)
	}
}

func TestFilterByText_NoMatch(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Title: "OK"},
		{ID: 2, Role: "txt", Title: "Hello"},
	}
	result := FilterByText(elements, "nonexistent")
	if len(result) != 0 {
		t.Errorf("expected 0 matches, got %d", len(result))
	}
}

func TestFilterByText_SubstringMatch(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Title: "Submit Review"},
	}
	result := FilterByText(elements, "mit")
	if len(result) != 1 {
		t.Errorf("expected 1 substring match, got %d", len(result))
	}
}

func TestFilterByText_RecursiveChildren(t *testing.T) {
	elements := []Element{
		{
			ID: 1, Role: "group", Title: "Form",
			Children: []Element{
				{ID: 2, Role: "btn", Title: "Submit"},
				{ID: 3, Role: "btn", Title: "Cancel"},
			},
		},
	}
	result := FilterByText(elements, "Submit")
	if len(result) != 1 {
		t.Fatalf("expected 1 top-level element (parent preserved), got %d", len(result))
	}
	if result[0].ID != 1 {
		t.Errorf("expected parent ID 1, got %d", result[0].ID)
	}
	if len(result[0].Children) != 1 {
		t.Errorf("expected 1 matching child, got %d", len(result[0].Children))
	}
	if result[0].Children[0].ID != 2 {
		t.Errorf("expected child ID 2, got %d", result[0].Children[0].ID)
	}
}

func TestFilterByText_ParentMatchesButChildDoesNot(t *testing.T) {
	elements := []Element{
		{
			ID: 1, Role: "group", Title: "Submit Section",
			Children: []Element{
				{ID: 2, Role: "txt", Title: "Description"},
			},
		},
	}
	result := FilterByText(elements, "Submit")
	if len(result) != 1 {
		t.Fatalf("expected 1 matching element, got %d", len(result))
	}
	// Parent matched; non-matching children should be pruned
	if len(result[0].Children) != 0 {
		t.Errorf("expected 0 children (non-matching pruned), got %d", len(result[0].Children))
	}
}

func TestFilterByText_PreservesIDs(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Title: "First"},
		{ID: 5, Role: "btn", Title: "Target"},
		{ID: 10, Role: "btn", Title: "Last"},
	}
	result := FilterByText(elements, "Target")
	if len(result) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result))
	}
	if result[0].ID != 5 {
		t.Errorf("expected ID 5 preserved, got %d", result[0].ID)
	}
}

func TestBoundsIntersect(t *testing.T) {
	tests := []struct {
		name string
		a, b [4]int
		want bool
	}{
		{"overlapping", [4]int{0, 0, 100, 100}, [4]int{50, 50, 100, 100}, true},
		{"adjacent_no_overlap", [4]int{0, 0, 100, 100}, [4]int{100, 0, 100, 100}, false},
		{"contained", [4]int{0, 0, 200, 200}, [4]int{50, 50, 10, 10}, true},
		{"no_overlap", [4]int{0, 0, 10, 10}, [4]int{20, 20, 10, 10}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boundsIntersect(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("boundsIntersect(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
