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

// AssertResult is the YAML output of an assert command.
type AssertResult struct {
	OK      bool         `yaml:"ok"                json:"ok"`
	Action  string       `yaml:"action"            json:"action"`
	Pass    bool         `yaml:"pass"              json:"pass"`
	Error   string       `yaml:"error,omitempty"   json:"error,omitempty"`
	Element *ElementInfo `yaml:"element,omitempty" json:"element,omitempty"`
}

var assertCmd = &cobra.Command{
	Use:   "assert",
	Short: "Assert a UI condition is met",
	Long: `Check that a UI element exists with expected properties.

Returns pass/fail with structured output and exit code 0 (pass) or 1 (fail).
Optionally polls with --timeout for conditions that take time to appear.`,
	RunE: runAssert,
}

func init() {
	rootCmd.AddCommand(assertCmd)
	assertCmd.Flags().String("app", "", "Scope to application")
	assertCmd.Flags().String("window", "", "Scope to window")
	assertCmd.Flags().Int("pid", 0, "Filter to a specific process by PID")
	assertCmd.Flags().Int("window-id", 0, "Filter to a specific window by system window ID")
	addTextTargetingFlags(assertCmd, "text", "Find element by title/value/description text")
	assertCmd.Flags().Int("id", 0, "Find element by ID")

	// Property assertions
	assertCmd.Flags().String("value", "", "Assert element value equals this string")
	assertCmd.Flags().String("value-contains", "", "Assert element value contains this substring")
	assertCmd.Flags().Bool("checked", false, "Assert element is selected/checked")
	assertCmd.Flags().Bool("unchecked", false, "Assert element is NOT selected/checked")
	assertCmd.Flags().Bool("disabled", false, "Assert element is disabled")
	assertCmd.Flags().Bool("enabled", false, "Assert element is enabled")
	assertCmd.Flags().Bool("is-focused", false, "Assert element has keyboard focus")
	assertCmd.Flags().Bool("gone", false, "Assert element does NOT exist")

	// Timing
	assertCmd.Flags().Int("timeout", 0, "Max seconds to poll (0 = single check, no polling)")
	assertCmd.Flags().Int("interval", 500, "Polling interval in milliseconds (default: 500)")
}

func runAssert(cmd *cobra.Command, args []string) error {
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
	text, roles, exact, scopeID := getTextTargetingFlags(cmd, "text")
	id, _ := cmd.Flags().GetInt("id")

	value, _ := cmd.Flags().GetString("value")
	valueContains, _ := cmd.Flags().GetString("value-contains")
	checked, _ := cmd.Flags().GetBool("checked")
	unchecked, _ := cmd.Flags().GetBool("unchecked")
	disabled, _ := cmd.Flags().GetBool("disabled")
	enabled, _ := cmd.Flags().GetBool("enabled")
	isFocused, _ := cmd.Flags().GetBool("is-focused")
	gone, _ := cmd.Flags().GetBool("gone")

	timeoutSec, _ := cmd.Flags().GetInt("timeout")
	intervalMs, _ := cmd.Flags().GetInt("interval")

	if text == "" && id == 0 {
		return fmt.Errorf("specify --text or --id to target an element")
	}
	if err := requireScope(appName, window, windowID, pid); err != nil {
		return err
	}

	hasValueCheck := cmd.Flags().Changed("value")

	opts := assertOptions{
		provider:      provider,
		appName:       appName,
		window:        window,
		windowID:      windowID,
		pid:           pid,
		text:          text,
		roles:         roles,
		exact:         exact,
		scopeID:       scopeID,
		id:            id,
		value:         value,
		hasValueCheck: hasValueCheck,
		valueContains: valueContains,
		checked:       checked,
		unchecked:     unchecked,
		disabled:      disabled,
		enabled:       enabled,
		isFocused:     isFocused,
		gone:          gone,
	}

	if timeoutSec > 0 {
		timeout := time.Duration(timeoutSec) * time.Second
		interval := time.Duration(intervalMs) * time.Millisecond
		deadline := time.Now().Add(timeout)

		for {
			result := checkAssert(opts)
			if result.Pass {
				return output.Print(result)
			}
			if time.Now().After(deadline) {
				_ = output.Print(result)
				return fmt.Errorf("assert failed: %s", result.Error)
			}
			time.Sleep(interval)
		}
	}

	result := checkAssert(opts)
	if result.Pass {
		return output.Print(result)
	}
	_ = output.Print(result)
	return fmt.Errorf("assert failed: %s", result.Error)
}

type assertOptions struct {
	provider      *platform.Provider
	appName       string
	window        string
	windowID      int
	pid           int
	text          string
	roles         string
	exact         bool
	scopeID       int
	id            int
	value         string
	hasValueCheck bool
	valueContains string
	checked       bool
	unchecked     bool
	disabled      bool
	enabled       bool
	isFocused     bool
	gone          bool
}

// checkAssert performs a single assertion check and returns the result.
func checkAssert(opts assertOptions) AssertResult {
	elem, err := findAssertElement(opts)

	if opts.gone {
		if err != nil || elem == nil {
			return AssertResult{OK: true, Action: "assert", Pass: true}
		}
		return AssertResult{
			OK:      false,
			Action:  "assert",
			Pass:    false,
			Error:   fmt.Sprintf("expected element to be gone but found: %s", describeElement(elem)),
			Element: elementInfoFromElement(elem),
		}
	}

	if err != nil {
		return AssertResult{
			OK:     false,
			Action: "assert",
			Pass:   false,
			Error:  err.Error(),
		}
	}

	// Element found — check property assertions
	if err := checkPropertyAssertions(elem, opts); err != nil {
		return AssertResult{
			OK:      false,
			Action:  "assert",
			Pass:    false,
			Error:   err.Error(),
			Element: elementInfoFromElement(elem),
		}
	}

	return AssertResult{
		OK:      true,
		Action:  "assert",
		Pass:    true,
		Element: elementInfoFromElement(elem),
	}
}

// findAssertElement locates the target element by text or ID.
func findAssertElement(opts assertOptions) (*model.Element, error) {
	if opts.provider == nil || opts.provider.Reader == nil {
		return nil, fmt.Errorf("reader not available on this platform")
	}

	if opts.text != "" {
		elem, _, err := resolveElementByText(opts.provider, opts.appName, opts.window, opts.windowID, opts.pid, opts.text, opts.roles, opts.exact, opts.scopeID)
		return elem, err
	}

	// By ID
	elements, err := opts.provider.Reader.ReadElements(platform.ReadOptions{
		App:      opts.appName,
		Window:   opts.window,
		WindowID: opts.windowID,
		PID:      opts.pid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read elements: %w", err)
	}
	elem := findElementByID(elements, opts.id)
	if elem == nil {
		return nil, fmt.Errorf("element with id %d not found", opts.id)
	}
	return elem, nil
}

// checkPropertyAssertions validates element properties against the assertion flags.
func checkPropertyAssertions(elem *model.Element, opts assertOptions) error {
	if opts.hasValueCheck {
		if elem.Value != opts.value {
			return fmt.Errorf("expected value %q but got %q", opts.value, elem.Value)
		}
	}
	if opts.valueContains != "" {
		if !strings.Contains(strings.ToLower(elem.Value), strings.ToLower(opts.valueContains)) {
			return fmt.Errorf("expected value to contain %q but got %q", opts.valueContains, elem.Value)
		}
	}
	if opts.checked {
		if !elem.Selected {
			return fmt.Errorf("expected element to be checked/selected but it is not")
		}
	}
	if opts.unchecked {
		if elem.Selected {
			return fmt.Errorf("expected element to be unchecked/unselected but it is selected")
		}
	}
	if opts.disabled {
		if elem.Enabled == nil || *elem.Enabled {
			return fmt.Errorf("expected element to be disabled but it is enabled")
		}
	}
	if opts.enabled {
		if elem.Enabled != nil && !*elem.Enabled {
			return fmt.Errorf("expected element to be enabled but it is disabled")
		}
	}
	if opts.isFocused {
		if !elem.Focused {
			return fmt.Errorf("expected element to be focused but it is not")
		}
	}
	return nil
}

// describeElement returns a brief human-readable description of an element.
func describeElement(elem *model.Element) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("id=%d", elem.ID))
	parts = append(parts, fmt.Sprintf("role=%s", elem.Role))
	if elem.Title != "" {
		parts = append(parts, fmt.Sprintf("title=%q", elem.Title))
	}
	if elem.Value != "" {
		parts = append(parts, fmt.Sprintf("value=%q", elem.Value))
	}
	return strings.Join(parts, " ")
}
