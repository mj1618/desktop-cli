package model

import "testing"

func TestMapRole_KnownRoles(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"AXButton", "btn"},
		{"AXStaticText", "txt"},
		{"AXLink", "lnk"},
		{"AXImage", "img"},
		{"AXTextField", "input"},
		{"AXTextArea", "input"},
		{"AXCheckBox", "chk"},
		{"AXRadioButton", "radio"},
		{"AXMenu", "menu"},
		{"AXMenuBar", "menu"},
		{"AXMenuItem", "menuitem"},
		{"AXTabGroup", "tab"},
		{"AXList", "list"},
		{"AXTable", "list"},
		{"AXRow", "row"},
		{"AXCell", "cell"},
		{"AXGroup", "group"},
		{"AXSplitGroup", "group"},
		{"AXScrollArea", "scroll"},
		{"AXToolbar", "toolbar"},
		{"AXWebArea", "web"},
		{"AXWindow", "window"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MapRole(tt.input)
			if got != tt.want {
				t.Errorf("MapRole(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapRole_UnknownFallback(t *testing.T) {
	unknowns := []string{"AXPopUpButton", "AXSlider", "AXProgressIndicator", "SomethingElse", ""}
	for _, role := range unknowns {
		got := MapRole(role)
		if got != "other" {
			t.Errorf("MapRole(%q) = %q, want %q", role, got, "other")
		}
	}
}
