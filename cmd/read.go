package cmd

import "github.com/spf13/cobra"

var readCmd = &cobra.Command{
	Use:   "read",
	Short: "Read the UI element tree and output as JSON",
	Long:  "Read the UI element tree from the OS accessibility layer and output as structured JSON.",
	RunE:  notImplemented("read"),
}

func init() {
	rootCmd.AddCommand(readCmd)
	readCmd.Flags().String("app", "", "Filter to a specific application by name")
	readCmd.Flags().String("window", "", "Filter to a specific window by title substring")
	readCmd.Flags().Int("window-id", 0, "Filter to a specific window by system window ID")
	readCmd.Flags().Int("pid", 0, "Filter to a specific process by PID")
	readCmd.Flags().Int("depth", 0, "Max depth to traverse (0 = unlimited)")
	readCmd.Flags().String("roles", "", "Comma-separated roles to include (e.g. \"button,textfield,link\")")
	readCmd.Flags().Bool("visible-only", true, "Only include visible/on-screen elements")
	readCmd.Flags().String("bbox", "", "Only include elements within bounding box (x,y,w,h)")
	readCmd.Flags().Bool("compact", false, "Ultra-compact output: flatten tree, minimal keys")
	readCmd.Flags().Bool("pretty", false, "Pretty-print JSON")
}
