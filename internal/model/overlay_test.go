package model

import "testing"

func TestDetectFrontmostOverlay_SubroleBased(t *testing.T) {
	elements := []Element{
		{
			ID: 1, Role: "window", Bounds: [4]int{0, 0, 1000, 800},
			Children: []Element{
				{ID: 2, Role: "group", Bounds: [4]int{0, 0, 1000, 800}},
				{ID: 3, Role: "group", Subrole: "AXDialog", Bounds: [4]int{200, 200, 600, 400},
					Children: []Element{
						{ID: 4, Role: "input", Title: "Subject", Bounds: [4]int{250, 300, 400, 30}},
						{ID: 5, Role: "btn", Title: "OK", Bounds: [4]int{450, 500, 100, 40}},
					},
				},
			},
		},
	}

	overlay := DetectFrontmostOverlay(elements)
	if overlay == nil {
		t.Fatal("expected overlay to be detected")
	}
	if overlay.ID != 3 {
		t.Errorf("expected overlay ID 3, got %d", overlay.ID)
	}
}

func TestDetectFrontmostOverlay_SheetSubrole(t *testing.T) {
	elements := []Element{
		{
			ID: 1, Role: "window", Bounds: [4]int{0, 0, 1000, 800},
			Children: []Element{
				{ID: 2, Role: "group", Bounds: [4]int{0, 0, 1000, 800}},
				{ID: 3, Role: "group", Subrole: "AXSheet", Bounds: [4]int{100, 100, 800, 500},
					Children: []Element{
						{ID: 4, Role: "input", Title: "File name", Bounds: [4]int{150, 200, 600, 30}},
					},
				},
			},
		},
	}

	overlay := DetectFrontmostOverlay(elements)
	if overlay == nil {
		t.Fatal("expected overlay to be detected")
	}
	if overlay.ID != 3 {
		t.Errorf("expected overlay ID 3, got %d", overlay.ID)
	}
}

func TestDetectFrontmostOverlay_NoOverlay(t *testing.T) {
	elements := []Element{
		{
			ID: 1, Role: "window", Bounds: [4]int{0, 0, 1000, 800},
			Children: []Element{
				{ID: 2, Role: "group", Bounds: [4]int{0, 0, 1000, 800},
					Children: []Element{
						{ID: 3, Role: "btn", Title: "Submit", Bounds: [4]int{400, 600, 100, 40}},
					},
				},
			},
		},
	}

	overlay := DetectFrontmostOverlay(elements)
	if overlay != nil {
		t.Errorf("expected no overlay, got element ID %d", overlay.ID)
	}
}

func TestDetectFrontmostOverlay_FocusBased(t *testing.T) {
	// Window has two children: main content and a smaller overlay with focus
	elements := []Element{
		{
			ID: 1, Role: "window", Bounds: [4]int{0, 0, 1000, 800},
			Children: []Element{
				{ID: 2, Role: "group", Bounds: [4]int{0, 0, 1000, 800},
					Children: []Element{
						{ID: 3, Role: "btn", Title: "Background Button", Bounds: [4]int{100, 100, 100, 40}},
					},
				},
				{ID: 4, Role: "group", Bounds: [4]int{250, 200, 500, 400},
					Children: []Element{
						{ID: 5, Role: "input", Title: "Dialog Input", Focused: true, Bounds: [4]int{300, 300, 400, 30}},
						{ID: 6, Role: "btn", Title: "OK", Bounds: [4]int{400, 500, 100, 40}},
					},
				},
			},
		},
	}

	overlay := DetectFrontmostOverlay(elements)
	if overlay == nil {
		t.Fatal("expected overlay to be detected via focus")
	}
	if overlay.ID != 4 {
		t.Errorf("expected overlay ID 4, got %d", overlay.ID)
	}
}

func TestDetectFrontmostOverlay_BoundsBased(t *testing.T) {
	// Window has two children: main content (full size) and a centered dialog (no subrole, no focus)
	elements := []Element{
		{
			ID: 1, Role: "window", Bounds: [4]int{0, 0, 1000, 800},
			Children: []Element{
				{ID: 2, Role: "group", Bounds: [4]int{0, 0, 1000, 800}},
				{ID: 3, Role: "group", Bounds: [4]int{300, 200, 400, 400},
					Children: []Element{
						{ID: 4, Role: "btn", Title: "Confirm", Bounds: [4]int{450, 500, 100, 40}},
					},
				},
			},
		},
	}

	overlay := DetectFrontmostOverlay(elements)
	if overlay == nil {
		t.Fatal("expected overlay to be detected via bounds")
	}
	if overlay.ID != 3 {
		t.Errorf("expected overlay ID 3, got %d", overlay.ID)
	}
}

func TestDetectFrontmostOverlay_SingleChild(t *testing.T) {
	// Window has only one child — no overlay possible
	elements := []Element{
		{
			ID: 1, Role: "window", Bounds: [4]int{0, 0, 1000, 800},
			Children: []Element{
				{ID: 2, Role: "group", Bounds: [4]int{0, 0, 1000, 800},
					Children: []Element{
						{ID: 3, Role: "btn", Title: "Submit", Bounds: [4]int{400, 600, 100, 40}},
					},
				},
			},
		},
	}

	overlay := DetectFrontmostOverlay(elements)
	if overlay != nil {
		t.Errorf("expected no overlay for single child, got element ID %d", overlay.ID)
	}
}

func TestDetectFrontmostOverlay_NestedSubrole(t *testing.T) {
	// Dialog subrole one level deeper (grandchild of window)
	elements := []Element{
		{
			ID: 1, Role: "window", Bounds: [4]int{0, 0, 1000, 800},
			Children: []Element{
				{ID: 2, Role: "group", Bounds: [4]int{0, 0, 1000, 800},
					Children: []Element{
						{ID: 3, Role: "group", Subrole: "AXDialog", Bounds: [4]int{200, 200, 600, 400},
							Children: []Element{
								{ID: 4, Role: "btn", Title: "OK", Bounds: [4]int{450, 500, 100, 40}},
							},
						},
					},
				},
			},
		},
	}

	overlay := DetectFrontmostOverlay(elements)
	if overlay == nil {
		t.Fatal("expected nested overlay to be detected")
	}
	if overlay.ID != 3 {
		t.Errorf("expected overlay ID 3, got %d", overlay.ID)
	}
}

func TestContainsFocused(t *testing.T) {
	el := Element{
		ID: 1, Role: "group",
		Children: []Element{
			{ID: 2, Role: "input", Focused: true},
		},
	}
	if !containsFocused(&el) {
		t.Error("expected containsFocused to return true")
	}

	el2 := Element{
		ID: 3, Role: "group",
		Children: []Element{
			{ID: 4, Role: "btn"},
		},
	}
	if containsFocused(&el2) {
		t.Error("expected containsFocused to return false")
	}
}

func TestIsOverlaySized(t *testing.T) {
	win := &Element{Bounds: [4]int{0, 0, 1000, 800}}

	// Clearly smaller dialog
	dialog := &Element{Bounds: [4]int{200, 200, 600, 400}}
	if !isOverlaySized(dialog, win) {
		t.Error("expected dialog to be overlay-sized")
	}

	// Full-size element (not an overlay)
	full := &Element{Bounds: [4]int{0, 0, 1000, 800}}
	if isOverlaySized(full, win) {
		t.Error("expected full-size element not to be overlay-sized")
	}
}

func TestIsCentered(t *testing.T) {
	win := &Element{Bounds: [4]int{0, 0, 1000, 800}}

	// Centered dialog
	centered := &Element{Bounds: [4]int{300, 200, 400, 400}}
	if !isCentered(centered, win) {
		t.Error("expected dialog to be centered")
	}

	// Off-center element
	offcenter := &Element{Bounds: [4]int{0, 0, 200, 200}}
	if isCentered(offcenter, win) {
		t.Error("expected off-center element not to be centered")
	}
}
