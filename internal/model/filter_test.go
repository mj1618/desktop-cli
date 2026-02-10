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

func TestFilterByFocused_NoFocused(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Title: "OK"},
		{ID: 2, Role: "txt", Title: "Hello"},
	}
	result := FilterByFocused(elements)
	if len(result) != 0 {
		t.Errorf("expected 0 elements when nothing is focused, got %d", len(result))
	}
}

func TestFilterByFocused_TopLevel(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Title: "OK"},
		{ID: 2, Role: "input", Title: "Search", Focused: true},
		{ID: 3, Role: "txt", Title: "Hello"},
	}
	result := FilterByFocused(elements)
	if len(result) != 1 {
		t.Fatalf("expected 1 focused element, got %d", len(result))
	}
	if result[0].ID != 2 {
		t.Errorf("expected ID 2, got %d", result[0].ID)
	}
	if !result[0].Focused {
		t.Error("expected Focused to be true")
	}
}

func TestFilterByFocused_NestedChild(t *testing.T) {
	elements := []Element{
		{
			ID: 1, Role: "window",
			Children: []Element{
				{
					ID: 2, Role: "group",
					Children: []Element{
						{ID: 3, Role: "input", Title: "Subject", Focused: true},
						{ID: 4, Role: "btn", Title: "Send"},
					},
				},
			},
		},
	}
	result := FilterByFocused(elements)
	if len(result) != 1 {
		t.Fatalf("expected 1 top-level element (ancestor preserved), got %d", len(result))
	}
	if result[0].ID != 1 {
		t.Errorf("expected ancestor ID 1, got %d", result[0].ID)
	}
	if len(result[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result[0].Children))
	}
	if result[0].Children[0].ID != 2 {
		t.Errorf("expected child ID 2, got %d", result[0].Children[0].ID)
	}
	if len(result[0].Children[0].Children) != 1 {
		t.Fatalf("expected 1 grandchild, got %d", len(result[0].Children[0].Children))
	}
	if result[0].Children[0].Children[0].ID != 3 {
		t.Errorf("expected grandchild ID 3, got %d", result[0].Children[0].Children[0].ID)
	}
}

func TestFilterByFocused_PreservesFields(t *testing.T) {
	elements := []Element{
		{ID: 5, Role: "input", Title: "Email", Value: "test@example.com", Focused: true, Bounds: [4]int{100, 200, 300, 20}},
	}
	result := FilterByFocused(elements)
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	el := result[0]
	if el.ID != 5 || el.Role != "input" || el.Title != "Email" || el.Value != "test@example.com" {
		t.Errorf("fields not preserved: %+v", el)
	}
	if el.Bounds != [4]int{100, 200, 300, 20} {
		t.Errorf("bounds not preserved: %v", el.Bounds)
	}
}

func TestFilterByFocused_PrunesNonFocusedSiblings(t *testing.T) {
	elements := []Element{
		{
			ID: 1, Role: "group",
			Children: []Element{
				{ID: 2, Role: "input", Title: "Name"},
				{ID: 3, Role: "input", Title: "Email", Focused: true},
				{ID: 4, Role: "btn", Title: "Submit"},
			},
		},
	}
	result := FilterByFocused(elements)
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	if len(result[0].Children) != 1 {
		t.Fatalf("expected 1 child (only focused), got %d", len(result[0].Children))
	}
	if result[0].Children[0].ID != 3 {
		t.Errorf("expected focused child ID 3, got %d", result[0].Children[0].ID)
	}
}

// --- PruneEmptyGroups (tree mode) tests ---

func TestPruneEmptyGroups_RemovesAnonymousGroups(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "group", Bounds: [4]int{0, 0, 100, 100}}, // empty group — should be removed
		{ID: 2, Role: "btn", Title: "Submit", Bounds: [4]int{10, 10, 80, 30}},
	}
	result := PruneEmptyGroups(elements)
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	if result[0].ID != 2 {
		t.Errorf("expected ID 2 (btn), got %d", result[0].ID)
	}
}

func TestPruneEmptyGroups_KeepsGroupsWithTitle(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "group", Title: "Form Section", Bounds: [4]int{0, 0, 100, 100}},
		{ID: 2, Role: "group", Value: "selected", Bounds: [4]int{0, 0, 100, 100}},
		{ID: 3, Role: "group", Description: "Navigation panel", Bounds: [4]int{0, 0, 100, 100}},
	}
	result := PruneEmptyGroups(elements)
	if len(result) != 3 {
		t.Errorf("expected 3 elements (all groups have text), got %d", len(result))
	}
}

func TestPruneEmptyGroups_PromotesChildren(t *testing.T) {
	elements := []Element{
		{
			ID: 1, Role: "group", Bounds: [4]int{0, 0, 200, 200}, // empty, should be removed
			Children: []Element{
				{ID: 2, Role: "btn", Title: "Submit", Bounds: [4]int{10, 10, 80, 30}},
				{ID: 3, Role: "btn", Title: "Cancel", Bounds: [4]int{10, 50, 80, 30}},
			},
		},
	}
	result := PruneEmptyGroups(elements)
	if len(result) != 2 {
		t.Fatalf("expected 2 promoted children, got %d", len(result))
	}
	if result[0].Title != "Submit" || result[1].Title != "Cancel" {
		t.Errorf("unexpected titles: %q, %q", result[0].Title, result[1].Title)
	}
}

func TestPruneEmptyGroups_NestedEmptyGroups(t *testing.T) {
	// Deeply nested empty groups should all be collapsed
	elements := []Element{
		{
			ID: 1, Role: "window", Title: "Main",
			Children: []Element{
				{
					ID: 2, Role: "group", // empty
					Children: []Element{
						{
							ID: 3, Role: "group", // empty
							Children: []Element{
								{
									ID: 4, Role: "group", // empty
									Children: []Element{
										{ID: 5, Role: "input", Description: "Subject", Bounds: [4]int{10, 10, 200, 20}},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	result := PruneEmptyGroups(elements)
	if len(result) != 1 {
		t.Fatalf("expected 1 top-level element, got %d", len(result))
	}
	if result[0].Role != "window" {
		t.Errorf("expected window, got %s", result[0].Role)
	}
	// All three empty groups should be collapsed, leaving input as direct child of window
	if len(result[0].Children) != 1 {
		t.Fatalf("expected 1 child of window, got %d", len(result[0].Children))
	}
	if result[0].Children[0].Role != "input" {
		t.Errorf("expected input as child of window, got %s", result[0].Children[0].Role)
	}
}

func TestPruneEmptyGroups_RemovesOtherRole(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "other", Bounds: [4]int{0, 0, 100, 100}}, // empty "other" — should be removed
		{ID: 2, Role: "btn", Title: "OK", Bounds: [4]int{10, 10, 80, 30}},
	}
	result := PruneEmptyGroups(elements)
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	if result[0].ID != 2 {
		t.Errorf("expected ID 2, got %d", result[0].ID)
	}
}

func TestPruneEmptyGroups_PreservesNonGroupRoles(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "toolbar", Bounds: [4]int{0, 0, 200, 50}},   // not group/other — keep
		{ID: 2, Role: "scroll", Bounds: [4]int{0, 50, 200, 300}},  // not group/other — keep
		{ID: 3, Role: "web", Bounds: [4]int{0, 0, 200, 200}},      // not group/other — keep
	}
	result := PruneEmptyGroups(elements)
	if len(result) != 3 {
		t.Errorf("expected 3 elements (non-group roles preserved), got %d", len(result))
	}
}

func TestPruneEmptyGroups_NilInput(t *testing.T) {
	result := PruneEmptyGroups(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 elements for nil input, got %d", len(result))
	}
}

// --- PruneEmptyGroupsFlat tests ---

func TestPruneEmptyGroupsFlat_Basic(t *testing.T) {
	elements := []FlatElement{
		{ID: 1, Role: "group", Path: "window > group"},
		{ID: 2, Role: "group", Path: "window > group > group"},
		{ID: 3, Role: "input", Description: "Subject", Path: "window > group > group > input"},
	}
	result := PruneEmptyGroupsFlat(elements)
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	if result[0].ID != 3 {
		t.Errorf("expected ID 3, got %d", result[0].ID)
	}
	// Path should be preserved from original
	if result[0].Path != "window > group > group > input" {
		t.Errorf("expected original path preserved, got %q", result[0].Path)
	}
}

func TestPruneEmptyGroupsFlat_KeepsGroupsWithText(t *testing.T) {
	elements := []FlatElement{
		{ID: 1, Role: "group", Title: "Navigation", Path: "window > group"},
		{ID: 2, Role: "group", Path: "window > group > group"}, // empty — removed
		{ID: 3, Role: "btn", Title: "Back", Path: "window > group > group > btn"},
	}
	result := PruneEmptyGroupsFlat(elements)
	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(result))
	}
	if result[0].ID != 1 || result[1].ID != 3 {
		t.Errorf("unexpected IDs: %d, %d", result[0].ID, result[1].ID)
	}
}

func TestPruneEmptyGroupsFlat_NilInput(t *testing.T) {
	result := PruneEmptyGroupsFlat(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 elements for nil input, got %d", len(result))
	}
}

// --- FilterFlatByText tests ---

func TestFilterFlatByText_EmptyText(t *testing.T) {
	elements := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Path: "window > btn"},
		{ID: 2, Role: "txt", Title: "Hello", Path: "window > txt"},
	}
	result := FilterFlatByText(elements, "")
	if len(result) != 2 {
		t.Errorf("expected 2 elements for empty text, got %d", len(result))
	}
}

func TestFilterFlatByText_MatchesOnly(t *testing.T) {
	elements := []FlatElement{
		{ID: 1, Role: "window", Title: "Main", Path: "window"},
		{ID: 2, Role: "group", Path: "window > group"},
		{ID: 3, Role: "group", Path: "window > group > group"},
		{ID: 4, Role: "btn", Title: "Submit", Path: "window > group > group > btn"},
		{ID: 5, Role: "btn", Title: "Cancel", Path: "window > group > group > btn"},
	}
	result := FilterFlatByText(elements, "Submit")
	if len(result) != 1 {
		t.Fatalf("expected 1 matching element, got %d", len(result))
	}
	if result[0].ID != 4 {
		t.Errorf("expected ID 4, got %d", result[0].ID)
	}
}

func TestFilterFlatByText_CaseInsensitive(t *testing.T) {
	elements := []FlatElement{
		{ID: 1, Role: "btn", Title: "SUBMIT"},
		{ID: 2, Role: "btn", Title: "cancel"},
	}
	result := FilterFlatByText(elements, "submit")
	if len(result) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result))
	}
	if result[0].ID != 1 {
		t.Errorf("expected ID 1, got %d", result[0].ID)
	}
}

func TestFilterFlatByText_MatchesValueAndDescription(t *testing.T) {
	elements := []FlatElement{
		{ID: 1, Role: "input", Value: "hello world"},
		{ID: 2, Role: "img", Description: "hello icon"},
		{ID: 3, Role: "btn", Title: "OK"},
	}
	result := FilterFlatByText(elements, "hello")
	if len(result) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(result))
	}
	if result[0].ID != 1 || result[1].ID != 2 {
		t.Errorf("expected IDs 1 and 2, got %d and %d", result[0].ID, result[1].ID)
	}
}

func TestFilterFlatByText_NoMatch(t *testing.T) {
	elements := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK"},
		{ID: 2, Role: "txt", Title: "Hello"},
	}
	result := FilterFlatByText(elements, "nonexistent")
	if len(result) != 0 {
		t.Errorf("expected 0 matches, got %d", len(result))
	}
}

func TestFilterFlatByText_PreservesPath(t *testing.T) {
	elements := []FlatElement{
		{ID: 1, Role: "btn", Title: "Submit", Path: "window > group > group > btn"},
	}
	result := FilterFlatByText(elements, "Submit")
	if len(result) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result))
	}
	if result[0].Path != "window > group > group > btn" {
		t.Errorf("expected path preserved, got %q", result[0].Path)
	}
}

func TestFilterFlatByText_NilInput(t *testing.T) {
	result := FilterFlatByText(nil, "test")
	if len(result) != 0 {
		t.Errorf("expected 0 elements for nil input, got %d", len(result))
	}
}

// --- FindFirstByText tests ---

func TestFindFirstByText_Found(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "group", Title: "Toolbar"},
		{ID: 2, Role: "group", Title: "Results", Children: []Element{
			{ID: 3, Role: "lnk", Title: "First Result"},
			{ID: 4, Role: "lnk", Title: "Second Result"},
		}},
		{ID: 5, Role: "btn", Title: "Next"},
	}
	result := FindFirstByText(elements, "Results")
	if result == nil {
		t.Fatal("expected to find element, got nil")
	}
	if result.ID != 2 {
		t.Errorf("expected ID 2, got %d", result.ID)
	}
	if len(result.Children) != 2 {
		t.Errorf("expected 2 children preserved, got %d", len(result.Children))
	}
}

func TestFindFirstByText_NotFound(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Title: "OK"},
	}
	result := FindFirstByText(elements, "nonexistent")
	if result != nil {
		t.Errorf("expected nil, got element ID %d", result.ID)
	}
}

func TestFindFirstByText_NestedMatch(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "window", Title: "Main", Children: []Element{
			{ID: 2, Role: "group", Children: []Element{
				{ID: 3, Role: "list", Title: "Search Results", Children: []Element{
					{ID: 4, Role: "row", Title: "Item 1"},
					{ID: 5, Role: "row", Title: "Item 2"},
				}},
			}},
		}},
	}
	result := FindFirstByText(elements, "Search Results")
	if result == nil {
		t.Fatal("expected to find nested element, got nil")
	}
	if result.ID != 3 {
		t.Errorf("expected ID 3, got %d", result.ID)
	}
}

func TestFindFirstByText_CaseInsensitive(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "group", Title: "RESULTS"},
	}
	result := FindFirstByText(elements, "results")
	if result == nil {
		t.Fatal("expected to find element, got nil")
	}
	if result.ID != 1 {
		t.Errorf("expected ID 1, got %d", result.ID)
	}
}

func TestFindFirstByText_ReturnsFirstMatch(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "group", Title: "Results A"},
		{ID: 2, Role: "group", Title: "Results B"},
	}
	result := FindFirstByText(elements, "Results")
	if result == nil {
		t.Fatal("expected to find element, got nil")
	}
	if result.ID != 1 {
		t.Errorf("expected first match ID 1, got %d", result.ID)
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
