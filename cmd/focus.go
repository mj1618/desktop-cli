package cmd

import (
	"fmt"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// FocusResult is the YAML output of a successful focus.
type FocusResult struct {
	OK     bool   `yaml:"ok"               json:"ok"`
	Action string `yaml:"action"           json:"action"`
	App    string `yaml:"app,omitempty"    json:"app,omitempty"`
	Window string `yaml:"window,omitempty" json:"window,omitempty"`
	PID    int    `yaml:"pid,omitempty"    json:"pid,omitempty"`
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

	return output.Print(FocusResult{
		OK:     true,
		Action: "focus",
		App:    appName,
		Window: window,
		PID:    pid,
	})
}
