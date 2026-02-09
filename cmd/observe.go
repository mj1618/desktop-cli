package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

var observeCmd = &cobra.Command{
	Use:   "observe",
	Short: "Watch for UI changes and stream diffs as JSONL",
	Long: `Continuously poll the UI element tree and emit changes (added, removed, modified elements) as JSONL to stdout.

Each line is a JSON object representing one change event. No output is emitted when the UI is stable.
This is far more token-efficient than repeatedly calling 'read' — only changes are reported.

Output is always JSONL regardless of the --format flag.

Use Ctrl+C or --duration to stop observing.`,
	RunE: runObserve,
}

func init() {
	rootCmd.AddCommand(observeCmd)
	observeCmd.Flags().String("app", "", "Scope to application (required unless --pid or --window-id is set)")
	observeCmd.Flags().String("window", "", "Scope to window by title substring")
	observeCmd.Flags().Int("window-id", 0, "Scope to window by system ID")
	observeCmd.Flags().Int("pid", 0, "Scope to process by PID")
	observeCmd.Flags().Int("depth", 0, "Max depth to traverse (0 = unlimited)")
	observeCmd.Flags().String("roles", "", "Comma-separated roles to include (e.g. \"btn,input\")")
	observeCmd.Flags().Int("interval", 1000, "Polling interval in milliseconds")
	observeCmd.Flags().Int("duration", 0, "Max seconds to observe (0 = until Ctrl+C)")
	observeCmd.Flags().Bool("ignore-bounds", false, "Ignore element position changes")
	observeCmd.Flags().Bool("ignore-focus", false, "Ignore focus changes")
}

func runObserve(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.Reader == nil {
		return fmt.Errorf("reader not available on this platform")
	}

	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")
	windowID, _ := cmd.Flags().GetInt("window-id")
	pid, _ := cmd.Flags().GetInt("pid")
	depth, _ := cmd.Flags().GetInt("depth")
	rolesStr, _ := cmd.Flags().GetString("roles")
	intervalMs, _ := cmd.Flags().GetInt("interval")
	durationSec, _ := cmd.Flags().GetInt("duration")
	ignoreBounds, _ := cmd.Flags().GetBool("ignore-bounds")
	ignoreFocus, _ := cmd.Flags().GetBool("ignore-focus")

	if appName == "" && pid == 0 && windowID == 0 {
		return fmt.Errorf("--app, --pid, or --window-id is required to scope observation")
	}

	var roles []string
	if rolesStr != "" {
		for _, r := range strings.Split(rolesStr, ",") {
			roles = append(roles, strings.TrimSpace(r))
		}
	}

	readOpts := platform.ReadOptions{
		App:      appName,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
		Depth:    depth,
		Roles:    roles,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)

	interval := time.Duration(intervalMs) * time.Millisecond
	var deadline time.Time
	if durationSec > 0 {
		deadline = time.Now().Add(time.Duration(durationSec) * time.Second)
	}
	start := time.Now()

	// Initial read to establish baseline
	elements, err := provider.Reader.ReadElements(readOpts)
	if err != nil {
		return fmt.Errorf("initial read failed: %w", err)
	}
	prevFlat := model.FlattenElements(elements)

	// Emit snapshot event
	enc.Encode(map[string]interface{}{
		"type":  "snapshot",
		"ts":    time.Now().Unix(),
		"count": len(prevFlat),
	})

	eventCount := 0

	// Poll loop
	for {
		if durationSec > 0 && time.Now().After(deadline) {
			break
		}

		time.Sleep(interval)

		elements, err := provider.Reader.ReadElements(readOpts)
		if err != nil {
			enc.Encode(map[string]interface{}{
				"type":  "error",
				"ts":    time.Now().Unix(),
				"error": err.Error(),
			})
			continue
		}

		currFlat := model.FlattenElements(elements)
		changes := model.DiffElements(prevFlat, currFlat)

		for _, change := range changes {
			if change.Type == model.ChangeChanged {
				if ignoreBounds {
					delete(change.Changes, "b")
				}
				if ignoreFocus {
					delete(change.Changes, "f")
				}
				if len(change.Changes) == 0 {
					continue
				}
			}
			enc.Encode(change)
			eventCount++
		}

		prevFlat = currFlat
	}

	// Emit done event
	elapsed := time.Since(start)
	enc.Encode(map[string]interface{}{
		"type":    "done",
		"ts":      time.Now().Unix(),
		"elapsed": fmt.Sprintf("%.1fs", elapsed.Seconds()),
		"events":  eventCount,
	})

	return nil
}
