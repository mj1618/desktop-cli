package model

import "testing"

func TestHasWebContent_Empty(t *testing.T) {
	if HasWebContent(nil) {
		t.Error("nil input should return false")
	}
	if HasWebContent([]Element{}) {
		t.Error("empty input should return false")
	}
}

func TestHasWebContent_TopLevel(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "window"},
		{ID: 2, Role: "web"},
	}
	if !HasWebContent(elements) {
		t.Error("should detect top-level web role")
	}
}

func TestHasWebContent_Nested(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "window", Children: []Element{
			{ID: 2, Role: "group", Children: []Element{
				{ID: 3, Role: "web"},
			}},
		}},
	}
	if !HasWebContent(elements) {
		t.Error("should detect nested web role")
	}
}

func TestHasWebContent_NoWeb(t *testing.T) {
	elements := []Element{
		{ID: 1, Role: "window", Children: []Element{
			{ID: 2, Role: "group"},
			{ID: 3, Role: "btn", Title: "OK"},
		}},
	}
	if HasWebContent(elements) {
		t.Error("should return false when no web role exists")
	}
}

func TestExpandRolesForWeb_NoWeb(t *testing.T) {
	roles := []string{"input", "btn"}
	result, expanded := ExpandRolesForWeb(roles, false)
	if expanded {
		t.Error("should not expand when hasWeb is false")
	}
	if len(result) != 2 {
		t.Errorf("expected 2 roles, got %d", len(result))
	}
}

func TestExpandRolesForWeb_WithInputNoOther(t *testing.T) {
	roles := []string{"input", "btn"}
	result, expanded := ExpandRolesForWeb(roles, true)
	if !expanded {
		t.Error("should expand when hasWeb is true and input present without other")
	}
	hasOther := false
	for _, r := range result {
		if r == "other" {
			hasOther = true
		}
	}
	if !hasOther {
		t.Error("expanded roles should include 'other'")
	}
}

func TestExpandRolesForWeb_AlreadyHasOther(t *testing.T) {
	roles := []string{"input", "other", "btn"}
	result, expanded := ExpandRolesForWeb(roles, true)
	if expanded {
		t.Error("should not expand when 'other' already present")
	}
	if len(result) != 3 {
		t.Errorf("expected 3 roles, got %d", len(result))
	}
}

func TestExpandRolesForWeb_NoInput(t *testing.T) {
	roles := []string{"btn", "lnk"}
	_, expanded := ExpandRolesForWeb(roles, true)
	if expanded {
		t.Error("should not expand when 'input' not in roles")
	}
}
