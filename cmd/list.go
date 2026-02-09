package cmd

import "github.com/spf13/cobra"

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available windows and applications",
	Long:  "List running applications or open windows with their app name, title, PID, and bounds.",
	RunE:  notImplemented("list"),
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().Bool("apps", false, "List running applications")
	listCmd.Flags().Bool("windows", false, "List all windows (default)")
	listCmd.Flags().Int("pid", 0, "Filter windows by PID")
	listCmd.Flags().String("app", "", "Filter windows by app name")
}
