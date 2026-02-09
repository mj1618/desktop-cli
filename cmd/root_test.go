package cmd

import (
	"testing"
)

func TestRootCommand_HasSubcommands(t *testing.T) {
	expected := []string{"read", "list", "click", "type", "focus", "scroll", "drag", "screenshot"}
	commands := rootCmd.Commands()

	found := make(map[string]bool)
	for _, c := range commands {
		found[c.Name()] = true
	}

	for _, name := range expected {
		if !found[name] {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

func TestRootCommand_Version(t *testing.T) {
	if rootCmd.Version == "" {
		t.Error("root command version should be set")
	}
}
