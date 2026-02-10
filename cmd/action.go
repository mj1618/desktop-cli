package cmd

import (
	"fmt"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// ActionResult is the YAML output of a successful action command.
type ActionResult struct {
	OK          bool          `yaml:"ok"                     json:"ok"`
	Action      string        `yaml:"action"                 json:"action"`
	ID          int           `yaml:"id"                     json:"id"`
	Name        string        `yaml:"name"                   json:"name"`
	Target      *ElementInfo  `yaml:"target,omitempty"       json:"target,omitempty"`
	Verified    *bool         `yaml:"verified,omitempty"     json:"verified,omitempty"`
	Retried     *bool         `yaml:"retried,omitempty"      json:"retried,omitempty"`
	RetryMethod string        `yaml:"retry_method,omitempty" json:"retry_method,omitempty"`
	RetryReason string        `yaml:"retry_reason,omitempty" json:"retry_reason,omitempty"`
	Display     []ElementInfo `yaml:"display,omitempty"      json:"display,omitempty"`
	State       string        `yaml:"state,omitempty"        json:"state,omitempty"`
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
	addRefFlag(actionCmd)
	actionCmd.Flags().Bool("no-display", false, "Skip collecting display elements in the response")
	addPostReadFlags(actionCmd)
	addVerifyFlags(actionCmd)
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
	text, roles, exact, scopeID := getTextTargetingFlags(cmd, "text")

	hasID := cmd.Flags().Changed("id")
	hasText := text != ""
	ref, _ := cmd.Flags().GetString("ref")
	hasRef := ref != ""

	if !hasID && !hasText && !hasRef {
		return fmt.Errorf("specify --id, --text, or --ref to target an element")
	}

	if err := requireScope(appName, window, windowID, pid); err != nil {
		return err
	}

	vOpts := getVerifyFlags(cmd)
	prOpts := getPostReadOptions(cmd)
	var preSnapshot elementSnapshot
	var preActionTarget *ElementInfo

	// Resolve target and capture pre-action snapshot
	if hasRef && !hasID {
		elem, _, err := resolveElementByRef(provider, appName, window, windowID, pid, ref)
		if err != nil {
			return err
		}
		id = elem.ID
		preActionTarget = elementInfoFromElement(elem)
		if vOpts.Verify {
			preSnapshot = snapshotElement(elem)
		}
	} else if hasText && !hasID {
		elem, _, err := resolveElementByText(provider, appName, window, windowID, pid, text, roles, exact, scopeID)
		if err != nil {
			return err
		}
		id = elem.ID
		preActionTarget = elementInfoFromElement(elem)
		if vOpts.Verify {
			preSnapshot = snapshotElement(elem)
		}
	} else {
		preActionTarget = readElementByID(provider, appName, window, windowID, pid, id)
		if vOpts.Verify && preActionTarget != nil {
			// Re-read to get full element for snapshot
			if elements, readErr := provider.Reader.ReadElements(platform.ReadOptions{
				App: appName, Window: window, WindowID: windowID, PID: pid,
			}); readErr == nil {
				if el := findElementByID(elements, id); el != nil {
					preSnapshot = snapshotElement(el)
				}
			}
		}
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

	// Verification: action has no fallback chain (already the most reliable method)
	var vr verifyResult
	if vOpts.Verify && preSnapshot.Exists {
		vr = verifyAction(provider, preSnapshot, vOpts, appName, window, windowID, pid, nil, prOpts.MaxElements)
	}

	// Read display elements after the action (e.g. Calculator display)
	noDisplay, _ := cmd.Flags().GetBool("no-display")

	var targetBounds [4]int
	if preActionTarget != nil {
		targetBounds = preActionTarget.Bounds
	}

	var display []ElementInfo
	if !noDisplay && !prOpts.PostRead {
		display = readDisplayElements(provider, appName, window, windowID, pid, targetBounds)
	}

	// Post-read: include full UI state in agent format
	var state string
	if prOpts.PostRead {
		if vr.PostState != "" {
			state = vr.PostState
		} else {
			state = readPostActionState(provider, appName, window, windowID, pid, prOpts.Delay, prOpts.MaxElements)
		}
	}

	result := ActionResult{
		OK:      true,
		Action:  "action",
		ID:      id,
		Name:    action,
		Target:  preActionTarget,
		Display: display,
		State:   state,
	}
	if vOpts.Verify && preSnapshot.Exists {
		result.Verified = boolPtr(vr.Verified)
		if vr.Retried {
			result.Retried = boolPtr(true)
			result.RetryMethod = vr.RetryMethod
			result.RetryReason = vr.RetryReason
		}
	}

	return output.Print(result)
}
