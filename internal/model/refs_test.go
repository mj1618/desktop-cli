package model

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Search", "search"},
		{"Full Name", "full-name"},
		{"OK", "ok"},
		{"Submit Form", "submit-form"},
		{"Address and search bar", "address-and-search-bar"},
		{"hello---world", "hello-world"},
		{"  spaces  ", "spaces"},
		{"Special!@#$%Chars", "special-chars"},
		{"Inbox (23288 unread)", "inbox-23288-unread"},
		{"", ""},
	}
	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBestLabel(t *testing.T) {
	tests := []struct {
		name string
		el   Element
		want string
	}{
		{"title", Element{Title: "Search"}, "Search"},
		{"description", Element{Description: "Search field"}, "Search field"},
		{"title over description", Element{Title: "Search", Description: "Search field"}, "Search"},
		{"no label", Element{Value: "some value"}, ""},
		{"empty", Element{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bestLabel(tt.el)
			if got != tt.want {
				t.Errorf("bestLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateRefs_BasicToolbar(t *testing.T) {
	elements := []Element{
		{
			ID:   1,
			Role: "window",
			Children: []Element{
				{
					ID:   2,
					Role: "toolbar",
					Children: []Element{
						{ID: 3, Role: "btn", Title: "Back", Actions: []string{"press"}},
						{ID: 4, Role: "input", Title: "Search", Bounds: [4]int{200, 50, 800, 30}},
					},
				},
				{
					ID:   5,
					Role: "group",
					Children: []Element{
						{ID: 6, Role: "btn", Title: "Submit", Actions: []string{"press"}},
					},
				},
			},
		},
	}

	GenerateRefs(elements)

	// btn "Back" under toolbar → toolbar/back
	backBtn := findByID(elements, 3)
	if backBtn.Ref != "toolbar/back" {
		t.Errorf("Back button ref = %q, want %q", backBtn.Ref, "toolbar/back")
	}

	// input "Search" under toolbar → toolbar/search
	searchInput := findByID(elements, 4)
	if searchInput.Ref != "toolbar/search" {
		t.Errorf("Search input ref = %q, want %q", searchInput.Ref, "toolbar/search")
	}

	// btn "Submit" under anonymous group → submit (group skipped)
	submitBtn := findByID(elements, 6)
	if submitBtn.Ref != "submit" {
		t.Errorf("Submit button ref = %q, want %q", submitBtn.Ref, "submit")
	}
}

func TestGenerateRefs_Dialog(t *testing.T) {
	elements := []Element{
		{
			ID:   1,
			Role: "window",
			Children: []Element{
				{
					ID:      10,
					Role:    "group",
					Subrole: "AXDialog",
					Children: []Element{
						{ID: 11, Role: "input", Title: "Email", Bounds: [4]int{100, 140, 200, 30}},
						{ID: 12, Role: "btn", Title: "OK", Actions: []string{"press"}},
						{ID: 13, Role: "btn", Title: "Cancel", Actions: []string{"press"}},
					},
				},
			},
		},
	}

	GenerateRefs(elements)

	emailInput := findByID(elements, 11)
	if emailInput.Ref != "dialog/email" {
		t.Errorf("Email input ref = %q, want %q", emailInput.Ref, "dialog/email")
	}

	okBtn := findByID(elements, 12)
	if okBtn.Ref != "dialog/ok" {
		t.Errorf("OK button ref = %q, want %q", okBtn.Ref, "dialog/ok")
	}

	cancelBtn := findByID(elements, 13)
	if cancelBtn.Ref != "dialog/cancel" {
		t.Errorf("Cancel button ref = %q, want %q", cancelBtn.Ref, "dialog/cancel")
	}
}

func TestGenerateRefs_Duplicates(t *testing.T) {
	elements := []Element{
		{
			ID:      1,
			Role:    "group",
			Subrole: "AXDialog",
			Children: []Element{
				{ID: 2, Role: "btn", Title: "OK", Actions: []string{"press"}},
				{ID: 3, Role: "btn", Title: "OK", Actions: []string{"press"}},
			},
		},
	}

	GenerateRefs(elements)

	ok1 := findByID(elements, 2)
	ok2 := findByID(elements, 3)
	if ok1.Ref != "dialog/ok.1" {
		t.Errorf("First OK button ref = %q, want %q", ok1.Ref, "dialog/ok.1")
	}
	if ok2.Ref != "dialog/ok.2" {
		t.Errorf("Second OK button ref = %q, want %q", ok2.Ref, "dialog/ok.2")
	}
}

func TestGenerateRefs_SkipsScrollAndWeb(t *testing.T) {
	elements := []Element{
		{
			ID:   1,
			Role: "window",
			Children: []Element{
				{
					ID:   2,
					Role: "scroll",
					Children: []Element{
						{
							ID:   3,
							Role: "web",
							Children: []Element{
								{
									ID:   4,
									Role: "group",
									Children: []Element{
										{ID: 5, Role: "btn", Title: "Login", Actions: []string{"press"}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	GenerateRefs(elements)

	loginBtn := findByID(elements, 5)
	// window, scroll, web, and unlabeled group all skipped → just "login"
	if loginBtn.Ref != "login" {
		t.Errorf("Login button ref = %q, want %q", loginBtn.Ref, "login")
	}
}

func TestGenerateRefs_ListLandmark(t *testing.T) {
	elements := []Element{
		{
			ID:   1,
			Role: "window",
			Children: []Element{
				{
					ID:   2,
					Role: "list",
					Children: []Element{
						{
							ID:   3,
							Role: "row",
							Children: []Element{
								{ID: 4, Role: "btn", Title: "Delete", Actions: []string{"press"}},
							},
						},
					},
				},
			},
		},
	}

	GenerateRefs(elements)

	deleteBtn := findByID(elements, 4)
	if deleteBtn.Ref != "list/delete" {
		t.Errorf("Delete button ref = %q, want %q", deleteBtn.Ref, "list/delete")
	}
}

func TestGenerateRefs_NonInteractiveSkipped(t *testing.T) {
	elements := []Element{
		{
			ID:   1,
			Role: "window",
			Children: []Element{
				{ID: 2, Role: "group", Title: "Sidebar"},
				// txt with no value and no press action → not interesting
				{ID: 3, Role: "txt", Title: "Label"},
				// txt with value → display text, gets ref
				{ID: 4, Role: "txt", Title: "Result", Value: "42"},
			},
		},
	}

	GenerateRefs(elements)

	group := findByID(elements, 2)
	if group.Ref != "" {
		t.Errorf("Non-interactive group should have no ref, got %q", group.Ref)
	}

	label := findByID(elements, 3)
	if label.Ref != "" {
		t.Errorf("txt without value should have no ref, got %q", label.Ref)
	}

	result := findByID(elements, 4)
	if result.Ref != "result" {
		t.Errorf("Display text ref = %q, want %q", result.Ref, "result")
	}
}

func TestFindElementByRef_ExactMatch(t *testing.T) {
	elements := []Element{
		{
			ID:   1,
			Role: "toolbar",
			Children: []Element{
				{ID: 2, Role: "btn", Title: "Back", Ref: "toolbar/back", Actions: []string{"press"}},
				{ID: 3, Role: "input", Title: "Search", Ref: "toolbar/search"},
			},
		},
	}

	el, err := FindElementByRef(elements, "toolbar/back")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if el.ID != 2 {
		t.Errorf("got ID %d, want 2", el.ID)
	}
}

func TestFindElementByRef_PartialMatch(t *testing.T) {
	elements := []Element{
		{
			ID:   1,
			Role: "toolbar",
			Children: []Element{
				{ID: 2, Role: "btn", Title: "Back", Ref: "toolbar/back", Actions: []string{"press"}},
				{ID: 3, Role: "input", Title: "Search", Ref: "toolbar/search"},
			},
		},
	}

	el, err := FindElementByRef(elements, "back")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if el.ID != 2 {
		t.Errorf("got ID %d, want 2", el.ID)
	}
}

func TestFindElementByRef_NotFound(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "btn", Title: "OK", Ref: "ok"},
	}

	_, err := FindElementByRef(elements, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ref")
	}
}

func TestFindElementByRef_Ambiguous(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "toolbar", Ref: "", Children: []Element{
			{ID: 2, Role: "btn", Title: "Submit", Ref: "toolbar/submit"},
		}},
		{ID: 3, Role: "group", Subrole: "AXDialog", Ref: "", Children: []Element{
			{ID: 4, Role: "btn", Title: "Submit", Ref: "dialog/submit"},
		}},
	}

	_, err := FindElementByRef(elements, "submit")
	if err == nil {
		t.Fatal("expected error for ambiguous ref")
	}
}

// findByID is a test helper that recursively finds an element by ID.
func findByID(elements []Element, id int) *Element {
	for i := range elements {
		if elements[i].ID == id {
			return &elements[i]
		}
		if found := findByID(elements[i].Children, id); found != nil {
			return found
		}
	}
	return nil
}
