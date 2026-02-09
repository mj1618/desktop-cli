package cmd

import "github.com/spf13/cobra"

var typeCmd = &cobra.Command{
	Use:   "type [text]",
	Short: "Type text or press key combinations",
	Long:  "Type text into the focused element or press key combinations. Text can be passed as a positional argument or via --text.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  notImplemented("type"),
}

func init() {
	rootCmd.AddCommand(typeCmd)
	typeCmd.Flags().String("text", "", "Text to type (alternative to positional arg)")
	typeCmd.Flags().String("key", "", "Key combination (e.g. \"cmd+c\", \"ctrl+shift+t\", \"enter\", \"tab\")")
	typeCmd.Flags().Int("delay", 0, "Delay between keystrokes in ms")
	typeCmd.Flags().Int("id", 0, "Focus element by ID first, then type")
	typeCmd.Flags().String("app", "", "Scope to application (used with --id)")
	typeCmd.Flags().String("window", "", "Scope to window (used with --id)")
}
