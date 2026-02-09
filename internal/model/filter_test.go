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
