package cmd

import (
	"fmt"
	"time"

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// SetValueResult is the output of a successful set-value command.
type SetValueResult struct {
	OK          bool   `yaml:"ok"                     json:"ok"`
	Action      string `yaml:"action"                 json:"action"`
	ID          int    `yaml:"id"                     json:"id"`
	Value       string `yaml:"value"                  json:"value"`
	Attribute   string `yaml:"attribute"               json:"attribute"`
	Verified    *bool  `yaml:"verified,omitempty"     json:"verified,omitempty"`
	Retried     *bool  `yaml:"retried,omitempty"      json:"retried,omitempty"`
	RetryMethod string `yaml:"retry_method,omitempty" json:"retry_method,omitempty"`
	RetryReason string `yaml:"retry_reason,omitempty" json:"retry_reason,omitempty"`
	State       string `yaml:"state,omitempty"        json:"state,omitempty"`
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
	addRefFlag(setValueCmd)
	addPostReadFlags(setValueCmd)
	addVerifyFlags(setValueCmd)
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
	var resolvedElem *model.Element

	// Resolve ref or text to element ID if needed
	if hasRef && !hasID {
		elem, _, err := resolveElementByRef(provider, appName, window, windowID, pid, ref)
		if err != nil {
			return err
		}
		id = elem.ID
		resolvedElem = elem
	} else if hasText && !hasID {
		elem, _, err := resolveElementByText(provider, appName, window, windowID, pid, text, roles, exact, scopeID)
		if err != nil {
			return err
		}
		id = elem.ID
		resolvedElem = elem
	} else if hasID {
		// Re-read to get full element for verify snapshot
		if provider.Reader != nil {
			if elements, readErr := provider.Reader.ReadElements(platform.ReadOptions{
				App: appName, Window: window, WindowID: windowID, PID: pid,
			}); readErr == nil {
				resolvedElem = findElementByID(elements, id)
			}
		}
	}

	if vOpts.Verify && resolvedElem != nil {
		preSnapshot = snapshotElement(resolvedElem)
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

	// Verification: retry with type (keystroke simulation) as fallback
	var vr verifyResult
	if vOpts.Verify && preSnapshot.Exists {
		var fallbacks []fallbackAction
		if provider.Inputter != nil && resolvedElem != nil && attribute == "value" {
			elemBounds := resolvedElem.Bounds
			fallbacks = append(fallbacks, fallbackAction{
				Method: "type",
				Execute: func() error {
					// Click to focus, select all, then type the value
					cx := elemBounds[0] + elemBounds[2]/2
					cy := elemBounds[1] + elemBounds[3]/2
					if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
						return err
					}
					time.Sleep(50 * time.Millisecond)
					if err := provider.Inputter.KeyCombo([]string{"cmd", "a"}); err != nil {
						return err
					}
					time.Sleep(30 * time.Millisecond)
					return provider.Inputter.TypeText(value, 0)
				},
			})
		}
		vr = verifyAction(provider, preSnapshot, vOpts, appName, window, windowID, pid, fallbacks, prOpts.MaxElements)
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

	result := SetValueResult{
		OK:        true,
		Action:    "set-value",
		ID:        id,
		Value:     value,
		Attribute: attribute,
		State:     state,
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
