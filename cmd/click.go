package cmd

import "github.com/spf13/cobra"

var clickCmd = &cobra.Command{
	Use:   "click",
	Short: "Click on an element or at coordinates",
	Long:  "Click on a UI element by ID (requires preceding read) or at absolute screen coordinates.",
	RunE:  notImplemented("click"),
}

func init() {
	rootCmd.AddCommand(clickCmd)
	clickCmd.Flags().Int("id", 0, "Click element by ID (requires preceding read)")
	clickCmd.Flags().Int("x", 0, "Click at absolute X screen coordinate")
	clickCmd.Flags().Int("y", 0, "Click at absolute Y screen coordinate")
	clickCmd.Flags().String("button", "left", "Mouse button: left, right, middle")
	clickCmd.Flags().Bool("double", false, "Double-click")
	clickCmd.Flags().String("app", "", "Scope to application (used with --id)")
	clickCmd.Flags().String("window", "", "Scope to window (used with --id)")
}
