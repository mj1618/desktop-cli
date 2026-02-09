package cmd

import "github.com/spf13/cobra"

var scrollCmd = &cobra.Command{
	Use:   "scroll",
	Short: "Scroll within a window or element",
	Long:  "Scroll up, down, left, or right within a window or specific element.",
	RunE:  notImplemented("scroll"),
}

func init() {
	rootCmd.AddCommand(scrollCmd)
	scrollCmd.Flags().String("direction", "", "Scroll direction: up, down, left, right")
	scrollCmd.Flags().Int("amount", 3, "Number of scroll clicks")
	scrollCmd.Flags().Int("x", 0, "Scroll at specific X coordinate")
	scrollCmd.Flags().Int("y", 0, "Scroll at specific Y coordinate")
	scrollCmd.Flags().Int("id", 0, "Scroll within element by ID")
	scrollCmd.Flags().String("app", "", "Scope to application")
	scrollCmd.Flags().String("window", "", "Scope to window")
}
