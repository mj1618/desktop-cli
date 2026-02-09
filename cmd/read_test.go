package cmd

import (
	"testing"
)

func TestReadCommand_Flags(t *testing.T) {
	flags := readCmd.Flags()

	tests := []struct {
		name     string
		flagType string
	}{
		{"app", "string"},
		{"window", "string"},
		{"window-id", "int"},
		{"pid", "int"},
		{"depth", "int"},
		{"roles", "string"},
		{"visible-only", "bool"},
		{"bbox", "string"},
		{"compact", "bool"},
		{"pretty", "bool"},
		{"text", "string"},
		{"flat", "bool"},
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

func TestReadCommand_IsRegistered(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Name() == "read" {
			return
		}
	}
	t.Error("read command not registered on root")
}
