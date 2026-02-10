package cmd

import (
	"fmt"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// SetValueResult is the output of a successful set-value command.
type SetValueResult struct {
	OK        bool   `yaml:"ok"                json:"ok"`
	Action    string `yaml:"action"            json:"action"`
	ID        int    `yaml:"id"                json:"id"`
	Value     string `yaml:"value"             json:"value"`
	Attribute string `yaml:"attribute"         json:"attribute"`
	State     string `yaml:"state,omitempty"   json:"state,omitempty"`
}

var setValueCmd = &cobra.Command{
	Use:   "set-value",
	Short: "Set the value of a UI element directly",
	Long: `Set an accessibility attribute value directly on a UI element by ID.

This sets the element's value without simulating keystrokes or mouse events.
Common use cases:
  - Set text field contents instantly (faster than type for long text)
  - Set slider positions to specific values
  - Set checkbox/toggle state without toggling

The --attribute flag controls which attribute to set:
  value     - Element value: text content, slider position, etc. (default)
  selected  - Selection state (true/false)
  focused   - Focus state (true/false)

Unlike 'type', this does NOT simulate keystrokes — it sets the value directly
via the accessibility API, which is faster and more reliable.`,
	RunE: runSetValue,
}

func init() {
	rootCmd.AddCommand(setValueCmd)
	setValueCmd.Flags().Int("id", 0, "Element ID from read output")
	setValueCmd.Flags().String("value", "", "Value to set (required)")
	setValueCmd.Flags().String("attribute", "value", "Attribute to set: value (default), selected, focused")
	setValueCmd.Flags().String("app", "", "Scope to application")
	setValueCmd.Flags().String("window", "", "Scope to window")
	setValueCmd.Flags().Int("window-id", 0, "Scope to window by system ID")
	setValueCmd.Flags().Int("pid", 0, "Scope to process by PID")
	addTextTargetingFlags(setValueCmd, "text", "Find element by text and set its value (case-insensitive match on title/value/description)")
	addPostReadFlags(setValueCmd)
}

func runSetValue(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.ValueSetter == nil {
		return fmt.Errorf("set-value not supported on this platform")
	}

	id, _ := cmd.Flags().GetInt("id")
	value, _ := cmd.Flags().GetString("value")
	attribute, _ := cmd.Flags().GetString("attribute")
	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")
	windowID, _ := cmd.Flags().GetInt("window-id")
	pid, _ := cmd.Flags().GetInt("pid")
	text, roles, exact, scopeID := getTextTargetingFlags(cmd, "text")

	hasID := cmd.Flags().Changed("id")
	hasText := text != ""

	if !hasID && !hasText {
		return fmt.Errorf("specify --id or --text to target an element")
	}

	if err := requireScope(appName, window, windowID, pid); err != nil {
		return err
	}

	// Resolve text to element ID if needed
	if hasText && !hasID {
		elem, _, err := resolveElementByText(provider, appName, window, windowID, pid, text, roles, exact, scopeID)
		if err != nil {
			return err
		}
		id = elem.ID
	}

	opts := platform.SetValueOptions{
		App:       appName,
		Window:    window,
		WindowID:  windowID,
		PID:       pid,
		ID:        id,
		Value:     value,
		Attribute: attribute,
	}

	err = provider.ValueSetter.SetValue(opts)
	if err != nil {
		return err
	}

	// Post-read: include full UI state in agent format
	postRead, postReadDelay := getPostReadFlags(cmd)
	var state string
	if postRead {
		state = readPostActionState(provider, appName, window, windowID, pid, postReadDelay)
	}

	return output.Print(SetValueResult{
		OK:        true,
		Action:    "set-value",
		ID:        id,
		Value:     value,
		Attribute: attribute,
		State:     state,
	})
}
