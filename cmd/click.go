package cmd

import (
	"fmt"

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// ClickResult is the YAML output of a successful click.
type ClickResult struct {
	OK          bool          `yaml:"ok"                     json:"ok"`
	Action      string        `yaml:"action"                 json:"action"`
	X           int           `yaml:"x"                      json:"x"`
	Y           int           `yaml:"y"                      json:"y"`
	Button      string        `yaml:"button"                 json:"button"`
	Count       int           `yaml:"count"                  json:"count"`
	Verified    *bool         `yaml:"verified,omitempty"     json:"verified,omitempty"`
	Retried     *bool         `yaml:"retried,omitempty"      json:"retried,omitempty"`
	RetryMethod string        `yaml:"retry_method,omitempty" json:"retry_method,omitempty"`
	RetryReason string        `yaml:"retry_reason,omitempty" json:"retry_reason,omitempty"`
	Display     []ElementInfo `yaml:"display,omitempty"      json:"display,omitempty"`
	State       string        `yaml:"state,omitempty"        json:"state,omitempty"`
}

var clickCmd = &cobra.Command{
	Use:   "click",
	Short: "Click on an element or at coordinates",
	Long:  "Click on a UI element by ID (requires preceding read) or at absolute screen coordinates.",
	RunE:  runClick,
}

func init() {
	rootCmd.AddCommand(clickCmd)
	clickCmd.Flags().Int("id", 0, "Click element by ID (re-reads element tree)")
	clickCmd.Flags().Int("x", 0, "Click at absolute X screen coordinate")
	clickCmd.Flags().Int("y", 0, "Click at absolute Y screen coordinate")
	clickCmd.Flags().String("button", "left", "Mouse button: left, right, middle")
	clickCmd.Flags().Bool("double", false, "Double-click")
	clickCmd.Flags().String("app", "", "Scope to application (used with --id or --text)")
	clickCmd.Flags().String("window", "", "Scope to window (used with --id or --text)")
	addTextTargetingFlags(clickCmd, "text", "Find and click element by text (case-insensitive match on title/value/description)")
	addRefFlag(clickCmd)
	clickCmd.Flags().Bool("near", false, "Click nearest interactive element to the text match (useful when text labels are not themselves clickable)")
	clickCmd.Flags().String("near-direction", "", "Search direction for --near: left, right, above, below (default: prefer left, then any)")
	clickCmd.Flags().Bool("no-display", false, "Skip collecting display elements in the response")
	addPostReadFlags(clickCmd)
	addVerifyFlags(clickCmd)
}

func runClick(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}

	id, _ := cmd.Flags().GetInt("id")
	x, _ := cmd.Flags().GetInt("x")
	y, _ := cmd.Flags().GetInt("y")
	buttonStr, _ := cmd.Flags().GetString("button")
	double, _ := cmd.Flags().GetBool("double")
	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")

	button, err := platform.ParseMouseButton(buttonStr)
	if err != nil {
		return err
	}

	count := 1
	if double {
		count = 2
	}

	hasCoords := cmd.Flags().Changed("x") || cmd.Flags().Changed("y")
	hasID := cmd.Flags().Changed("id")
	text, roles, exact, scopeID := getTextTargetingFlags(cmd, "text")
	hasText := text != ""
	ref, _ := cmd.Flags().GetString("ref")
	hasRef := ref != ""

	near, _ := cmd.Flags().GetBool("near")
	nearDirection, _ := cmd.Flags().GetString("near-direction")

	vOpts := getVerifyFlags(cmd)
	prOpts := getPostReadOptions(cmd)
	var preSnapshot elementSnapshot
	var resolvedElem *model.Element // kept for verify fallback (action press)
	var targetBounds [4]int         // bounds of clicked element for display proximity

	if hasRef {
		// Stable ref targeting mode
		if appName == "" && window == "" {
			return fmt.Errorf("--ref requires --app or --window to scope the element lookup")
		}
		elem, _, err := resolveElementByRef(provider, appName, window, 0, 0, ref)
		if err != nil {
			return err
		}
		targetBounds = elem.Bounds
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
		if vOpts.Verify {
			resolvedElem = elem
			preSnapshot = snapshotElement(elem)
		}
	} else if hasText {
		// Text targeting mode: find element by text content
		if appName == "" && window == "" {
			return fmt.Errorf("--text requires --app or --window to scope the element lookup")
		}

		if near {
			// --near mode: get ALL text matches, pick the best one for proximity search.
			// This avoids the ambiguity error from resolveElementByText and prefers
			// matches in the main content area over sidebar/preview matches.
			allMatches, allElements, err := resolveAllTextMatches(provider, appName, window, 0, 0, text, roles, exact, scopeID)
			if err != nil {
				return err
			}

			elem := pickBestNearMatch(allElements, allMatches)
			targetBounds = elem.Bounds
			if vOpts.Verify {
				resolvedElem = elem
				preSnapshot = snapshotElement(elem)
			}

			// Find the nearest interactive element to the text match
			nearest := findNearestInteractiveElement(allElements, elem, nearDirection)
			if nearest != nil {
				elem = nearest
				targetBounds = elem.Bounds
				x = elem.Bounds[0] + elem.Bounds[2]/2
				y = elem.Bounds[1] + elem.Bounds[3]/2
				if vOpts.Verify {
					resolvedElem = elem
					preSnapshot = snapshotElement(elem)
				}
			} else {
				// No nearby interactive element found — use offset-based fallback
				x, y = nearFallbackOffset(elem, nearDirection)
			}
		} else {
			elem, _, err := resolveElementByText(provider, appName, window, 0, 0, text, roles, exact, scopeID)
			if err != nil {
				return err
			}

			targetBounds = elem.Bounds
			if vOpts.Verify {
				resolvedElem = elem
				preSnapshot = snapshotElement(elem)
			}
			x = elem.Bounds[0] + elem.Bounds[2]/2
			y = elem.Bounds[1] + elem.Bounds[3]/2
		}
	} else if hasID {
		// Element ID mode: re-read the element tree and find the element
		if appName == "" && window == "" {
			return fmt.Errorf("--id requires --app or --window to scope the element lookup")
		}
		if provider.Reader == nil {
			return fmt.Errorf("reader not available on this platform")
		}

		opts := platform.ReadOptions{
			App:    appName,
			Window: window,
		}
		elements, err := provider.Reader.ReadElements(opts)
		if err != nil {
			return err
		}

		elem := findElementByID(elements, id)
		if elem == nil {
			return fmt.Errorf("element with id %d not found", id)
		}

		targetBounds = elem.Bounds
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
		if vOpts.Verify {
			resolvedElem = elem
			preSnapshot = snapshotElement(elem)
		}
	} else if !hasCoords {
		return fmt.Errorf("specify --text, --ref, --id, or --x/--y coordinates")
	}

	if provider.Inputter == nil {
		return fmt.Errorf("input not available on this platform")
	}

	if err := provider.Inputter.Click(x, y, button, count); err != nil {
		return err
	}

	// Verification: check if UI changed, retry with fallbacks if not
	var vr verifyResult
	if vOpts.Verify && preSnapshot.Exists {
		var fallbacks []fallbackAction
		// Fallback 1: accessibility "press" action
		if provider.ActionPerformer != nil && resolvedElem != nil {
			elemID := resolvedElem.ID
			fallbacks = append(fallbacks, fallbackAction{
				Method: "action",
				Execute: func() error {
					return provider.ActionPerformer.PerformAction(platform.ActionOptions{
						App: appName, Window: window, ID: elemID, Action: "press",
					})
				},
			})
		}
		// Fallback 2: offset click (+2px)
		fallbacks = append(fallbacks, fallbackAction{
			Method: "offset-click",
			Execute: func() error {
				return provider.Inputter.Click(x+2, y+2, button, count)
			},
		})
		vr = verifyAction(provider, preSnapshot, vOpts, appName, window, 0, 0, fallbacks, prOpts.MaxElements)
	}

	// Read display elements after the click (e.g. Calculator display)
	noDisplay, _ := cmd.Flags().GetBool("no-display")

	var display []ElementInfo
	if !noDisplay && !prOpts.PostRead && (appName != "" || window != "") {
		display = readDisplayElements(provider, appName, window, 0, 0, targetBounds)
	}

	// Post-read: include full UI state in agent format
	var state string
	if prOpts.PostRead && (appName != "" || window != "") {
		if vr.PostState != "" {
			// Reuse the tree already read during verification
			state = vr.PostState
		} else {
			state = readPostActionState(provider, appName, window, 0, 0, prOpts.Delay, prOpts.MaxElements)
		}
	}

	result := ClickResult{
		OK:     true,
		Action: "click",
		X:      x,
		Y:      y,
		Button: buttonStr,
		Count:  count,
	}
	if vOpts.Verify && preSnapshot.Exists {
		result.Verified = boolPtr(vr.Verified)
		if vr.Retried {
			result.Retried = boolPtr(true)
			result.RetryMethod = vr.RetryMethod
			result.RetryReason = vr.RetryReason
		}
	}
	result.Display = display
	result.State = state

	return output.Print(result)
}
