package cmd

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Capture a screenshot",
	Long:  "Capture a screenshot of a window or the entire screen for vision model fallback.",
	RunE:  runScreenshot,
}

func init() {
	rootCmd.AddCommand(screenshotCmd)
	screenshotCmd.Flags().String("window", "", "Capture window by title substring")
	screenshotCmd.Flags().String("app", "", "Capture specific app's frontmost window")
	screenshotCmd.Flags().Int("window-id", 0, "Capture window by system ID")
	screenshotCmd.Flags().Int("pid", 0, "Capture frontmost window of this PID")
	screenshotCmd.Flags().String("output", "", "Output file path (default: stdout as base64)")
	screenshotCmd.Flags().String("format", "png", "Output format: png, jpg")
	screenshotCmd.Flags().Int("quality", 80, "JPEG quality 1-100")
	screenshotCmd.Flags().Float64("scale", 0.5, "Scale factor 0.1-1.0 (for token efficiency)")
	screenshotCmd.Flags().Bool("include-menubar", false, "Include macOS menu bar in app screenshots")
}

func runScreenshot(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.Screenshotter == nil {
		return fmt.Errorf("screenshot not supported on this platform")
	}

	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")
	windowID, _ := cmd.Flags().GetInt("window-id")
	pid, _ := cmd.Flags().GetInt("pid")
	output, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")
	quality, _ := cmd.Flags().GetInt("quality")
	scale, _ := cmd.Flags().GetFloat64("scale")
	includeMenuBar, _ := cmd.Flags().GetBool("include-menubar")

	opts := platform.ScreenshotOptions{
		App:            appName,
		Window:         window,
		WindowID:       windowID,
		PID:            pid,
		Format:         format,
		Quality:        quality,
		Scale:          scale,
		IncludeMenuBar: includeMenuBar,
	}

	data, err := provider.Screenshotter.CaptureWindow(opts)
	if err != nil {
		return err
	}

	// Output to file or stdout
	if output != "" {
		return os.WriteFile(output, data, 0644)
	}

	// Default: write to stdout as base64 for easy agent consumption
	encoder := base64.NewEncoder(base64.StdEncoding, os.Stdout)
	if _, err := encoder.Write(data); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	fmt.Println() // newline after base64
	return nil
}
