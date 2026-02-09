package cmd

import "github.com/spf13/cobra"

var focusCmd = &cobra.Command{
	Use:   "focus",
	Short: "Bring a window or application to the foreground",
	Long:  "Focus a window or application by name, title, window ID, or PID.",
	RunE:  notImplemented("focus"),
}

func init() {
	rootCmd.AddCommand(focusCmd)
	focusCmd.Flags().String("app", "", "Focus application by name")
	focusCmd.Flags().String("window", "", "Focus window by title substring")
	focusCmd.Flags().Int("window-id", 0, "Focus window by system ID")
	focusCmd.Flags().Int("pid", 0, "Focus application by PID")
}
