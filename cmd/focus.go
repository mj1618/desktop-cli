package cmd

import (
	"fmt"
	"time"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// FocusResult is the YAML output of a successful focus.
type FocusResult struct {
	OK          bool   `yaml:"ok"                      json:"ok"`
	Action      string `yaml:"action"                  json:"action"`
	App         string `yaml:"app,omitempty"           json:"app,omitempty"`
	Window      string `yaml:"window,omitempty"        json:"window,omitempty"`
	PID         int    `yaml:"pid,omitempty"           json:"pid,omitempty"`
	NewDocument bool   `yaml:"new_document,omitempty"  json:"new_document,omitempty"`
}

var focusCmd = &cobra.Command{
	Use:   "focus",
	Short: "Bring a window or application to the foreground",
	Long:  "Focus a window or application by name, title, window ID, or PID.",
	RunE:  runFocus,
}

func init() {
	rootCmd.AddCommand(focusCmd)
	focusCmd.Flags().String("app", "", "Focus application by name")
	focusCmd.Flags().String("window", "", "Focus window by title substring")
	focusCmd.Flags().Int("window-id", 0, "Focus window by system ID")
	focusCmd.Flags().Int("pid", 0, "Focus application by PID")
	focusCmd.Flags().Bool("new-document", false, "After focusing, dismiss any open dialog (Escape) and create a new document (Cmd+N)")
}

func runFocus(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}

	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")
	windowID, _ := cmd.Flags().GetInt("window-id")
	pid, _ := cmd.Flags().GetInt("pid")
	newDocument, _ := cmd.Flags().GetBool("new-document")

	if appName == "" && window == "" && windowID == 0 && pid == 0 {
		return fmt.Errorf("specify --app, --window, --window-id, or --pid")
	}

	if provider.WindowManager == nil {
		return fmt.Errorf("window management not available on this platform")
	}

	opts := platform.FocusOptions{
		App:      appName,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
	}

	if err := provider.WindowManager.FocusWindow(opts); err != nil {
		return err
	}

	// --new-document: dismiss any file-open dialog and create a blank document
	if newDocument {
		if provider.Inputter == nil {
			return fmt.Errorf("input simulation not available on this platform (required for --new-document)")
		}
		// Wait for the app to settle after focusing
		time.Sleep(300 * time.Millisecond)

		// Press Escape to dismiss any open dialog (e.g. file-open dialog in TextEdit)
		if err := provider.Inputter.KeyCombo([]string{"escape"}); err != nil {
			return fmt.Errorf("failed to dismiss dialog: %w", err)
		}
		time.Sleep(200 * time.Millisecond)

		// Press Cmd+N to create a new blank document
		if err := provider.Inputter.KeyCombo([]string{"cmd", "n"}); err != nil {
			return fmt.Errorf("failed to create new document: %w", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	return output.Print(FocusResult{
		OK:          true,
		Action:      "focus",
		App:         appName,
		Window:      window,
		PID:         pid,
		NewDocument: newDocument,
	})
}
