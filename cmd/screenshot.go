package cmd

import "github.com/spf13/cobra"

var screenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Capture a screenshot",
	Long:  "Capture a screenshot of a window or the entire screen for vision model fallback.",
	RunE:  notImplemented("screenshot"),
}

func init() {
	rootCmd.AddCommand(screenshotCmd)
	screenshotCmd.Flags().String("window", "", "Capture specific window")
	screenshotCmd.Flags().String("app", "", "Capture specific app's frontmost window")
	screenshotCmd.Flags().String("output", "", "Output file path (default: stdout as base64 or /tmp/screenshot.png)")
	screenshotCmd.Flags().String("format", "png", "Output format: png, jpg")
	screenshotCmd.Flags().Int("quality", 80, "JPEG quality 1-100")
	screenshotCmd.Flags().Float64("scale", 0.5, "Scale factor 0.1-1.0 (for token efficiency)")
}
