package cmd

import (
	"testing"
)

func TestListCommand_Flags(t *testing.T) {
	flags := listCmd.Flags()

	tests := []struct {
		name     string
		flagType string
	}{
		{"apps", "bool"},
		{"windows", "bool"},
		{"pid", "int"},
		{"app", "string"},
		{"pretty", "bool"},
	}

	for _, tt := range tests {
		f := flags.Lookup(tt.name)
		if f == nil {
			t.Errorf("expected flag %q not found", tt.name)
			continue
		}
		if f.Value.Type() != tt.flagType {
			t.Errorf("flag %q: expected type %q, got %q", tt.name, tt.flagType, f.Value.Type())
		}
	}
}

func TestListCommand_IsRegistered(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Name() == "list" {
			return
		}
	}
	t.Error("list command not registered on root")
}
