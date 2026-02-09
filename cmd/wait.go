package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// WaitResult is the YAML output of a wait command.
type WaitResult struct {
	OK      bool   `yaml:"ok"`
	Action  string `yaml:"action"`
	Elapsed string `yaml:"elapsed"`
	Match   string `yaml:"match,omitempty"`
	TimedOut bool  `yaml:"timed_out,omitempty"`
}

var waitCmd = &cobra.Command{
	Use:   "wait",
	Short: "Wait for a UI condition to be met",
	Long:  "Poll the UI element tree until a specified condition is met or timeout is reached.",
	RunE:  runWait,
}

func init() {
	rootCmd.AddCommand(waitCmd)
	waitCmd.Flags().String("app", "", "Scope to application")
	waitCmd.Flags().String("window", "", "Scope to window")
	waitCmd.Flags().Int("pid", 0, "Filter to a specific process by PID")
	waitCmd.Flags().Int("window-id", 0, "Filter to a specific window by system window ID")
	waitCmd.Flags().String("for-text", "", "Wait for element with this title or value text (substring match)")
	waitCmd.Flags().String("for-role", "", "Wait for element with this role (e.g. btn, input, lnk)")
	waitCmd.Flags().Int("for-id", 0, "Wait for element with this ID to exist")
	waitCmd.Flags().Bool("gone", false, "Invert: wait until the condition is NO LONGER true")
	waitCmd.Flags().Int("timeout", 30, "Max seconds to wait (default: 30)")
	waitCmd.Flags().Int("interval", 500, "Polling interval in milliseconds (default: 500)")
	waitCmd.Flags().Int("depth", 0, "Max depth to traverse (0 = unlimited)")
	waitCmd.Flags().String("roles", "", "Comma-separated roles to filter during read")
}

func runWait(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.Reader == nil {
		return fmt.Errorf("reader not available on this platform")
	}

	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")
	pid, _ := cmd.Flags().GetInt("pid")
	windowID, _ := cmd.Flags().GetInt("window-id")
	forText, _ := cmd.Flags().GetString("for-text")
	forRole, _ := cmd.Flags().GetString("for-role")
	forID, _ := cmd.Flags().GetInt("for-id")
	gone, _ := cmd.Flags().GetBool("gone")
	timeoutSec, _ := cmd.Flags().GetInt("timeout")
	intervalMs, _ := cmd.Flags().GetInt("interval")
	depth, _ := cmd.Flags().GetInt("depth")
	rolesStr, _ := cmd.Flags().GetString("roles")

	if forText == "" && forRole == "" && forID == 0 {
		return fmt.Errorf("specify at least one condition: --for-text, --for-role, or --for-id")
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
		PID:      pid,
		WindowID: windowID,
		Depth:    depth,
		Roles:    roles,
	}

	timeout := time.Duration(timeoutSec) * time.Second
	interval := time.Duration(intervalMs) * time.Millisecond
	deadline := time.Now().Add(timeout)
	start := time.Now()

	for {
		elements, err := provider.Reader.ReadElements(readOpts)
		if err != nil {
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout after %s (last error: %w)", timeout, err)
			}
			time.Sleep(interval)
			continue
		}

		matched := checkWaitCondition(elements, forText, forRole, forID)

		conditionMet := matched
		if gone {
			conditionMet = !matched
		}

		if conditionMet {
			elapsed := time.Since(start)
			matchDesc := describeCondition(forText, forRole, forID, gone)
			return output.PrintYAML(WaitResult{
				OK:      true,
				Action:  "wait",
				Elapsed: fmt.Sprintf("%.1fs", elapsed.Seconds()),
				Match:   matchDesc,
			})
		}

		if time.Now().After(deadline) {
			elapsed := time.Since(start)
			matchDesc := describeCondition(forText, forRole, forID, gone)
			// Print the result, then return an error for non-zero exit code
			_ = output.PrintYAML(WaitResult{
				OK:      false,
				Action:  "wait",
				Elapsed: fmt.Sprintf("%.1fs", elapsed.Seconds()),
				Match:   matchDesc,
				TimedOut: true,
			})
			return fmt.Errorf("timed out waiting for condition: %s", matchDesc)
		}

		time.Sleep(interval)
	}
}

// checkWaitCondition checks if any element in the tree matches the wait criteria.
func checkWaitCondition(elements []model.Element, forText, forRole string, forID int) bool {
	for _, elem := range elements {
		if matchesCondition(elem, forText, forRole, forID) {
			return true
		}
		if checkWaitCondition(elem.Children, forText, forRole, forID) {
			return true
		}
	}
	return false
}

// matchesCondition checks if a single element matches all specified criteria.
// When multiple criteria are given, ALL must match (AND logic).
func matchesCondition(elem model.Element, forText, forRole string, forID int) bool {
	if forID > 0 && elem.ID != forID {
		return false
	}
	if forRole != "" && elem.Role != forRole {
		return false
	}
	if forText != "" {
		textLower := strings.ToLower(forText)
		titleMatch := strings.Contains(strings.ToLower(elem.Title), textLower)
		valueMatch := strings.Contains(strings.ToLower(elem.Value), textLower)
		descMatch := strings.Contains(strings.ToLower(elem.Description), textLower)
		if !titleMatch && !valueMatch && !descMatch {
			return false
		}
	}
	return true
}

// describeCondition returns a human-readable description of what was waited for.
func describeCondition(forText, forRole string, forID int, gone bool) string {
	var parts []string
	if forRole != "" {
		parts = append(parts, fmt.Sprintf("role=%s", forRole))
	}
	if forText != "" {
		parts = append(parts, fmt.Sprintf("text=%q", forText))
	}
	if forID > 0 {
		parts = append(parts, fmt.Sprintf("id=%d", forID))
	}
	desc := strings.Join(parts, " ")
	if gone {
		desc += " (gone)"
	}
	return desc
}
