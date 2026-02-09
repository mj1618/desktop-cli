package cmd

import (
	"fmt"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available windows and applications",
	Long:  "List running applications or open windows with their app name, title, PID, and bounds.",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().Bool("apps", false, "List running applications")
	listCmd.Flags().Bool("windows", false, "List all windows (default)")
	listCmd.Flags().Int("pid", 0, "Filter windows by PID")
	listCmd.Flags().String("app", "", "Filter windows by app name")
	listCmd.Flags().Bool("pretty", false, "Pretty-print output (no-op for YAML)")
}

// appEntry is the YAML output for --apps mode.
type appEntry struct {
	App string `yaml:"app"`
	PID int    `yaml:"pid"`
}

func runList(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}

	apps, _ := cmd.Flags().GetBool("apps")
	pid, _ := cmd.Flags().GetInt("pid")
	appName, _ := cmd.Flags().GetString("app")

	if provider.Reader == nil {
		return fmt.Errorf("reader not available on this platform")
	}

	opts := platform.ListOptions{
		Apps: apps,
		PID:  pid,
		App:  appName,
	}

	windows, err := provider.Reader.ListWindows(opts)
	if err != nil {
		return err
	}

	if apps {
		// Aggregate to unique apps
		seen := make(map[string]bool)
		var entries []appEntry
		for _, w := range windows {
			if !seen[w.App] {
				seen[w.App] = true
				entries = append(entries, appEntry{App: w.App, PID: w.PID})
			}
		}
		if entries == nil {
			entries = []appEntry{}
		}
		return output.PrintYAML(entries)
	}

	return output.PrintYAML(windows)
}
