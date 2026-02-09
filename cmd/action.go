package cmd

import (
	"fmt"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// ActionResult is the YAML output of a successful action command.
type ActionResult struct {
	OK     bool         `yaml:"ok"                json:"ok"`
	Action string       `yaml:"action"            json:"action"`
	ID     int          `yaml:"id"                json:"id"`
	Name   string       `yaml:"name"              json:"name"`
	Target *ElementInfo `yaml:"target,omitempty"  json:"target,omitempty"`
}

var actionCmd = &cobra.Command{
	Use:   "action",
	Short: "Perform an accessibility action on a UI element",
	Long: `Execute an accessibility action directly on a UI element by ID.

Actions are the same as shown in the 'a' field of 'read' output:
  press      - Press/activate the element (buttons, links, menu items)
  cancel     - Cancel the current operation
  pick       - Pick/select (dropdowns, menus)
  increment  - Increase value (sliders, steppers)
  decrement  - Decrease value (sliders, steppers)
  confirm    - Confirm a dialog or selection
  showMenu   - Show context menu for the element
  raise      - Bring element/window to front

Unlike 'click', this does NOT simulate mouse events — it calls the accessibility
action directly on the element, which works even for off-screen or occluded elements.`,
	RunE: runAction,
}

func init() {
	rootCmd.AddCommand(actionCmd)
	actionCmd.Flags().Int("id", 0, "Element ID from read output")
	actionCmd.Flags().String("action", "press", "Action to perform (default: press)")
	actionCmd.Flags().String("app", "", "Scope to application")
	actionCmd.Flags().String("window", "", "Scope to window")
	actionCmd.Flags().Int("window-id", 0, "Scope to window by system ID")
	actionCmd.Flags().Int("pid", 0, "Scope to process by PID")
	addTextTargetingFlags(actionCmd, "text", "Find element by text and perform action (case-insensitive match on title/value/description)")
}

func runAction(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.ActionPerformer == nil {
		return fmt.Errorf("action not supported on this platform")
	}

	id, _ := cmd.Flags().GetInt("id")
	action, _ := cmd.Flags().GetString("action")
	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")
	windowID, _ := cmd.Flags().GetInt("window-id")
	pid, _ := cmd.Flags().GetInt("pid")
	text, roles := getTextTargetingFlags(cmd, "text")

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
		elem, _, err := resolveElementByText(provider, appName, window, windowID, pid, text, roles)
		if err != nil {
			return err
		}
		id = elem.ID
	}

	opts := platform.ActionOptions{
		App:      appName,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
		ID:       id,
		Action:   action,
	}

	if err := provider.ActionPerformer.PerformAction(opts); err != nil {
		return err
	}

	result := ActionResult{
		OK:     true,
		Action: "action",
		ID:     id,
		Name:   action,
	}

	// Re-read the target element to include its current state
	result.Target = readElementByID(provider, appName, window, windowID, pid, id)

	return output.Print(result)
}
