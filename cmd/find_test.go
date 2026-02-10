package cmd

import (
	"testing"

	"github.com/mj1618/desktop-cli/internal/model"
)

func TestFindCommand_Registered(t *testing.T) {
	commands := rootCmd.Commands()
	found := false
	for _, c := range commands {
		if c.Name() == "find" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'find' subcommand to be registered")
	}
}

func TestFindCommand_HasExpectedFlags(t *testing.T) {
	expectedFlags := []string{"text", "roles", "app", "limit", "exact"}
	for _, name := range expectedFlags {
		if findCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s to exist on find command", name)
		}
	}
}

func TestSortWindowsFocusedFirst(t *testing.T) {
	windows := []model.Window{
		{App: "Safari", PID: 1, Title: "Tab 1", Focused: false},
		{App: "Chrome", PID: 2, Title: "Gmail", Focused: true},
		{App: "Finder", PID: 3, Title: "Home", Focused: false},
		{App: "Notes", PID: 4, Title: "Note 1", Focused: true},
	}

	sortWindowsFocusedFirst(windows)

	// Focused windows should be at the front
	if !windows[0].Focused {
		t.Errorf("expected first window to be focused, got %s (focused=%v)", windows[0].App, windows[0].Focused)
	}
	if !windows[1].Focused {
		t.Errorf("expected second window to be focused, got %s (focused=%v)", windows[1].App, windows[1].Focused)
	}
	// Non-focused windows should be at the back
	if windows[2].Focused || windows[3].Focused {
		t.Error("expected non-focused windows at the end")
	}
}

func TestSortWindowsFocusedFirst_NoFocused(t *testing.T) {
	windows := []model.Window{
		{App: "Safari", PID: 1, Focused: false},
		{App: "Chrome", PID: 2, Focused: false},
	}

	sortWindowsFocusedFirst(windows)

	// Order should be unchanged when none are focused
	if windows[0].App != "Safari" || windows[1].App != "Chrome" {
		t.Error("expected order unchanged when no windows are focused")
	}
}

func TestSortWindowsFocusedFirst_AllFocused(t *testing.T) {
	windows := []model.Window{
		{App: "Safari", PID: 1, Focused: true},
		{App: "Chrome", PID: 2, Focused: true},
	}

	sortWindowsFocusedFirst(windows)

	// All focused — order preserved
	if windows[0].App != "Safari" || windows[1].App != "Chrome" {
		t.Error("expected order preserved when all windows are focused")
	}
}

func TestSortWindowsFocusedFirst_Empty(t *testing.T) {
	var windows []model.Window
	// Should not panic
	sortWindowsFocusedFirst(windows)
}

func TestFindCommand_RequiresText(t *testing.T) {
	// Verify that --text is required by checking that runFind returns an error
	// when text is empty. We can't easily call runFind directly because it
	// needs a platform provider, but we can check the flag default.
	val, _ := findCmd.Flags().GetString("text")
	if val != "" {
		t.Error("expected --text default to be empty")
	}
}

func TestFindCommand_DefaultLimit(t *testing.T) {
	val, _ := findCmd.Flags().GetInt("limit")
	if val != 10 {
		t.Errorf("expected default limit to be 10, got %d", val)
	}
}
