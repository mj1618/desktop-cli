package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// TypeResult is the YAML output of a successful type command.
type TypeResult struct {
	OK          bool          `yaml:"ok"                     json:"ok"`
	Action      string        `yaml:"action"                 json:"action"`
	Text        string        `yaml:"text,omitempty"         json:"text,omitempty"`
	Key         string        `yaml:"key,omitempty"          json:"key,omitempty"`
	Target      *ElementInfo  `yaml:"target,omitempty"       json:"target,omitempty"`
	Focused     *ElementInfo  `yaml:"focused,omitempty"      json:"focused,omitempty"`
	Warning     string        `yaml:"warning,omitempty"      json:"warning,omitempty"`
	Verified    *bool         `yaml:"verified,omitempty"     json:"verified,omitempty"`
	Retried     *bool         `yaml:"retried,omitempty"      json:"retried,omitempty"`
	RetryMethod string        `yaml:"retry_method,omitempty" json:"retry_method,omitempty"`
	RetryReason string        `yaml:"retry_reason,omitempty" json:"retry_reason,omitempty"`
	Display     []ElementInfo `yaml:"display,omitempty"      json:"display,omitempty"`
	State       string        `yaml:"state,omitempty"        json:"state,omitempty"`
}

var typeCmd = &cobra.Command{
	Use:   "type [text]",
	Short: "Type text or press key combinations",
	Long:  "Type text into the focused element or press key combinations. Text can be passed as a positional argument or via --text.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runType,
}

func init() {
	rootCmd.AddCommand(typeCmd)
	typeCmd.Flags().String("text", "", "Text to type (alternative to positional arg)")
	typeCmd.Flags().String("key", "", "Key combination (e.g. \"cmd+c\", \"ctrl+shift+t\", \"enter\", \"tab\")")
	typeCmd.Flags().Int("delay", 0, "Delay between keystrokes in ms")
	typeCmd.Flags().Int("id", 0, "Focus element by ID first, then type")
	typeCmd.Flags().String("app", "", "Scope to application (used with --id or --target)")
	typeCmd.Flags().String("window", "", "Scope to window (used with --id or --target)")
	addTextTargetingFlags(typeCmd, "target", "Find element by text and focus it before typing (case-insensitive match on title/value/description)")
	addRefFlag(typeCmd)
	typeCmd.Flags().Bool("no-display", false, "Skip collecting display elements in the response")
	addPostReadFlags(typeCmd)
	addVerifyFlags(typeCmd)
}

func runType(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.Inputter == nil {
		return fmt.Errorf("input simulation not available on this platform")
	}

	text, _ := cmd.Flags().GetString("text")
	key, _ := cmd.Flags().GetString("key")
	delayMs, _ := cmd.Flags().GetInt("delay")
	id, _ := cmd.Flags().GetInt("id")
	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")

	// Positional arg overrides --text flag
	if len(args) > 0 {
		text = args[0]
	}

	if text == "" && key == "" {
		return fmt.Errorf("specify --text, --key, or a positional text argument")
	}

	target, roles, exact, scopeID := getTextTargetingFlags(cmd, "target")
	hasTarget := target != ""
	ref, _ := cmd.Flags().GetString("ref")
	hasRef := ref != ""

	vOpts := getVerifyFlags(cmd)
	prOpts := getPostReadOptions(cmd)

	// Track whether we targeted a specific element
	hasTargetedElement := false
	var verifyElemID int // ID of element to snapshot for verification

	// We'll read the focused element AFTER typing to get fresh state that
	// reflects any dialog/modal that appeared and the actual typed value.
	var postFocused *ElementInfo

	// If --ref, --target, or --id specified, click the element first to focus it
	if hasRef {
		if appName == "" && window == "" {
			return fmt.Errorf("--ref requires --app or --window to scope the element lookup")
		}
		elem, _, err := resolveElementByRef(provider, appName, window, 0, 0, ref)
		if err != nil {
			return err
		}
		hasTargetedElement = true
		verifyElemID = elem.ID
		cx := elem.Bounds[0] + elem.Bounds[2]/2
		cy := elem.Bounds[1] + elem.Bounds[3]/2
		if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
			return fmt.Errorf("failed to focus element: %w", err)
		}
		time.Sleep(50 * time.Millisecond)
	} else if hasTarget {
		if appName == "" && window == "" {
			return fmt.Errorf("--target requires --app or --window to scope the element lookup")
		}
		elem, _, err := resolveElementByText(provider, appName, window, 0, 0, target, roles, exact, scopeID)
		if err != nil {
			return err
		}
		hasTargetedElement = true
		verifyElemID = elem.ID
		cx := elem.Bounds[0] + elem.Bounds[2]/2
		cy := elem.Bounds[1] + elem.Bounds[3]/2
		if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
			return fmt.Errorf("failed to focus element: %w", err)
		}
		time.Sleep(50 * time.Millisecond)
	} else if id > 0 {
		if appName == "" && window == "" {
			return fmt.Errorf("--id requires --app or --window to scope the element lookup")
		}
		if provider.Reader == nil {
			return fmt.Errorf("reader not available on this platform")
		}
		elements, err := provider.Reader.ReadElements(platform.ReadOptions{
			App:    appName,
			Window: window,
		})
		if err != nil {
			return fmt.Errorf("failed to read elements: %w", err)
		}
		elem := findElementByID(elements, id)
		if elem == nil {
			return fmt.Errorf("element with id %d not found", id)
		}
		hasTargetedElement = true
		verifyElemID = elem.ID
		cx := elem.Bounds[0] + elem.Bounds[2]/2
		cy := elem.Bounds[1] + elem.Bounds[3]/2
		if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
			return fmt.Errorf("failed to focus element: %w", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Snapshot the focused element for verification (after click-to-focus, before typing)
	var preSnapshot elementSnapshot
	if vOpts.Verify && hasTargetedElement && provider.Reader != nil {
		elements, readErr := provider.Reader.ReadElements(platform.ReadOptions{
			App: appName, Window: window,
		})
		if readErr == nil {
			if focused := findFocusedElementRaw(elements); focused != nil {
				preSnapshot = snapshotElement(focused)
				verifyElemID = focused.ID
			} else if el := findElementByID(elements, verifyElemID); el != nil {
				preSnapshot = snapshotElement(el)
			}
		}
	}

	// Type text first (if provided)
	if text != "" {
		// Calculator mode: when targeting Calculator with no specific element,
		// translate text into button presses since Calculator has no text input.
		if isCalculatorApp(appName) && !hasTargetedElement && provider.ActionPerformer != nil {
			if err := typeViaButtons(provider, appName, window, text); err != nil {
				return err
			}
		} else {
			if err := provider.Inputter.TypeText(text, delayMs); err != nil {
				return err
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Capture target/focused element info after typing but before the key press.
	// If a key like "tab" moves focus to another element, we still want
	// to report the original target (not the newly-focused element).
	// Reading AFTER typing ensures we get fresh state that reflects any
	// dialog/modal that appeared and the actual typed value.
	var targetInfo *ElementInfo
	if hasTargetedElement {
		targetInfo = readFocusedElement(provider, appName, window, 0, 0)
	} else if text != "" {
		postFocused = readFocusedElement(provider, appName, window, 0, 0)
	}

	// Then press key combo (if provided)
	if key != "" {
		keys := strings.Split(key, "+")
		if err := provider.Inputter.KeyCombo(keys); err != nil {
			return err
		}
		time.Sleep(80 * time.Millisecond)
	}

	// Verification: retry with set-value (direct value injection) as fallback
	var vr verifyResult
	if vOpts.Verify && preSnapshot.Exists {
		var fallbacks []fallbackAction
		if text != "" && provider.ValueSetter != nil {
			elemID := verifyElemID
			fallbacks = append(fallbacks, fallbackAction{
				Method: "set-value",
				Execute: func() error {
					return provider.ValueSetter.SetValue(platform.SetValueOptions{
						App: appName, Window: window,
						ID: elemID, Value: text, Attribute: "value",
					})
				},
			})
		}
		vr = verifyAction(provider, preSnapshot, vOpts, appName, window, 0, 0, fallbacks, prOpts.MaxElements)
	}

	// Build result
	result := TypeResult{
		OK:   true,
		Text: text,
		Key:  key,
	}

	// Determine action label
	switch {
	case text != "" && key != "":
		result.Action = "type+key"
	case key != "":
		result.Action = "key"
	default:
		result.Action = "type"
	}

	// Include target/focused element info.
	if hasTargetedElement {
		result.Target = targetInfo
	} else if key != "" {
		// Key-only or text+key: report the post-action focused element,
		// since the key (e.g. tab, enter) may have intentionally moved focus.
		result.Focused = readFocusedElement(provider, appName, window, 0, 0)
	} else {
		// Text-only without target: use the post-typing focused element
		// which reflects the actual state after text was entered (including
		// any dialog/modal that appeared and the current value).
		result.Focused = postFocused
	}

	// Warn when the focused element doesn't appear to be text-editable.
	// This catches cases where macOS reports focus on a non-input element
	// (e.g. a cell or group) even though text was typed into a hidden input.
	if text != "" && !hasTargetedElement && result.Focused != nil {
		if !editableRoles[result.Focused.Role] {
			result.Warning = "focused element does not appear to be editable — text may have been entered into a different field"
		}
	}

	if vOpts.Verify && preSnapshot.Exists {
		result.Verified = boolPtr(vr.Verified)
		if vr.Retried {
			result.Retried = boolPtr(true)
			result.RetryMethod = vr.RetryMethod
			result.RetryReason = vr.RetryReason
		}
	}

	// Include display elements (e.g. Calculator display) when app is scoped
	noDisplay, _ := cmd.Flags().GetBool("no-display")

	// Use target/focused element bounds for proximity-based display selection
	var displayTargetBounds [4]int
	if result.Target != nil {
		displayTargetBounds = result.Target.Bounds
	} else if result.Focused != nil {
		displayTargetBounds = result.Focused.Bounds
	}

	if !noDisplay && !prOpts.PostRead && (appName != "" || window != "") {
		result.Display = readDisplayElements(provider, appName, window, 0, 0, displayTargetBounds)
	}

	// Post-read: include full UI state in agent format
	if prOpts.PostRead && (appName != "" || window != "") {
		if vr.PostState != "" {
			result.State = vr.PostState
		} else {
			result.State = readPostActionState(provider, appName, window, 0, 0, prOpts.Delay, prOpts.MaxElements)
		}
	}

	return output.Print(result)
}

// calculatorButtonMap maps text characters to Calculator button titles.
var calculatorButtonMap = map[byte]string{
	'0': "0", '1': "1", '2': "2", '3': "3", '4': "4",
	'5': "5", '6': "6", '7': "7", '8': "8", '9': "9",
	'+': "Add", '-': "Subtract", '*': "Multiply", '/': "Divide",
	'=': "Equals", '.': "Point",
}

// isCalculatorApp returns true if the app name matches Calculator.
func isCalculatorApp(appName string) bool {
	return strings.EqualFold(appName, "calculator")
}

// typeViaButtons presses Calculator buttons for each character in text.
func typeViaButtons(provider *platform.Provider, appName, window, text string) error {
	for i := 0; i < len(text); i++ {
		btnTitle, ok := calculatorButtonMap[text[i]]
		if !ok {
			return fmt.Errorf("unsupported Calculator character: %c", text[i])
		}
		elem, _, err := resolveElementByText(provider, appName, window, 0, 0, btnTitle, "btn", false, 0)
		if err != nil {
			return fmt.Errorf("Calculator button %q not found for character %c: %w", btnTitle, text[i], err)
		}
		if err := provider.ActionPerformer.PerformAction(platform.ActionOptions{
			App:    appName,
			Window: window,
			ID:     elem.ID,
			Action: "press",
		}); err != nil {
			return fmt.Errorf("failed to press Calculator button %q: %w", btnTitle, err)
		}
		time.Sleep(30 * time.Millisecond)
	}
	return nil
}
