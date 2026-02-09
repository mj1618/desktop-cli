package cmd

import "github.com/spf13/cobra"

var dragCmd = &cobra.Command{
	Use:   "drag",
	Short: "Drag from one point to another",
	Long:  "Drag from one point to another using coordinates or element IDs.",
	RunE:  notImplemented("drag"),
}

func init() {
	rootCmd.AddCommand(dragCmd)
	dragCmd.Flags().Int("from-x", 0, "Start X coordinate")
	dragCmd.Flags().Int("from-y", 0, "Start Y coordinate")
	dragCmd.Flags().Int("to-x", 0, "End X coordinate")
	dragCmd.Flags().Int("to-y", 0, "End Y coordinate")
	dragCmd.Flags().Int("from-id", 0, "Start element (center)")
	dragCmd.Flags().Int("to-id", 0, "End element (center)")
	dragCmd.Flags().String("app", "", "Scope to application")
	dragCmd.Flags().String("window", "", "Scope to window")
}
