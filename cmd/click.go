package cmd

import (
	"fmt"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// ClickResult is the YAML output of a successful click.
type ClickResult struct {
	OK     bool   `yaml:"ok"     json:"ok"`
	Action string `yaml:"action" json:"action"`
	X      int    `yaml:"x"      json:"x"`
	Y      int    `yaml:"y"      json:"y"`
	Button string `yaml:"button" json:"button"`
	Count  int    `yaml:"count"  json:"count"`
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
	clickCmd.Flags().String("app", "", "Scope to application (used with --id)")
	clickCmd.Flags().String("window", "", "Scope to window (used with --id)")
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

	if id > 0 {
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

		// Compute center of bounding box
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	} else if x == 0 && y == 0 {
		return fmt.Errorf("specify --id or --x/--y coordinates")
	}

	if provider.Inputter == nil {
		return fmt.Errorf("input not available on this platform")
	}

	if err := provider.Inputter.Click(x, y, button, count); err != nil {
		return err
	}

	return output.Print(ClickResult{
		OK:     true,
		Action: "click",
		X:      x,
		Y:      y,
		Button: buttonStr,
		Count:  count,
	})
}
