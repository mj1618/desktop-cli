package cmd

import (
	"fmt"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// HoverResult is the YAML output of a successful hover.
type HoverResult struct {
	OK     bool         `yaml:"ok"                json:"ok"`
	Action string       `yaml:"action"            json:"action"`
	X      int          `yaml:"x"                 json:"x"`
	Y      int          `yaml:"y"                 json:"y"`
	Target *ElementInfo `yaml:"target,omitempty"   json:"target,omitempty"`
	State  string       `yaml:"state,omitempty"   json:"state,omitempty"`
}

var hoverCmd = &cobra.Command{
	Use:   "hover",
	Short: "Move the mouse cursor to an element or coordinates without clicking",
	Long:  "Move the mouse to a UI element by ID, text, or absolute coordinates without clicking. Useful for triggering hover-dependent UI like tooltips, row actions, and flyout menus.",
	RunE:  runHover,
}

func init() {
	rootCmd.AddCommand(hoverCmd)
	hoverCmd.Flags().Int("id", 0, "Hover over element by ID (re-reads element tree)")
	hoverCmd.Flags().Int("x", 0, "Hover at absolute X screen coordinate")
	hoverCmd.Flags().Int("y", 0, "Hover at absolute Y screen coordinate")
	hoverCmd.Flags().String("app", "", "Scope to application (used with --id or --text)")
	hoverCmd.Flags().String("window", "", "Scope to window (used with --id or --text)")
	addTextTargetingFlags(hoverCmd, "text", "Find and hover over element by text (case-insensitive match on title/value/description)")
	addRefFlag(hoverCmd)
	addPostReadFlags(hoverCmd)
}

func runHover(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}

	id, _ := cmd.Flags().GetInt("id")
	x, _ := cmd.Flags().GetInt("x")
	y, _ := cmd.Flags().GetInt("y")
	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")

	hasCoords := cmd.Flags().Changed("x") || cmd.Flags().Changed("y")
	hasID := cmd.Flags().Changed("id")
	text, roles, exact, scopeID := getTextTargetingFlags(cmd, "text")
	hasText := text != ""
	ref, _ := cmd.Flags().GetString("ref")
	hasRef := ref != ""

	var target *ElementInfo

	if hasRef {
		if appName == "" && window == "" {
			return fmt.Errorf("--ref requires --app or --window to scope the element lookup")
		}
		elem, _, err := resolveElementByRef(provider, appName, window, 0, 0, ref)
		if err != nil {
			return err
		}
		target = elementInfoFromElement(elem)
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	} else if hasText {
		if appName == "" && window == "" {
			return fmt.Errorf("--text requires --app or --window to scope the element lookup")
		}
		elem, _, err := resolveElementByText(provider, appName, window, 0, 0, text, roles, exact, scopeID)
		if err != nil {
			return err
		}
		target = elementInfoFromElement(elem)
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	} else if hasID {
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
		target = elementInfoFromElement(elem)
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	} else if !hasCoords {
		return fmt.Errorf("specify --text, --ref, --id, or --x/--y coordinates")
	}

	if provider.Inputter == nil {
		return fmt.Errorf("input not available on this platform")
	}

	if err := provider.Inputter.MoveMouse(x, y); err != nil {
		return err
	}

	// Post-read: include full UI state in agent format
	prOpts := getPostReadOptions(cmd)
	var state string
	if prOpts.PostRead && (appName != "" || window != "") {
		state = readPostActionState(provider, appName, window, 0, 0, prOpts.Delay, prOpts.MaxElements)
	}

	return output.Print(HoverResult{
		OK:     true,
		Action: "hover",
		X:      x,
		Y:      y,
		Target: target,
		State:  state,
	})
}
