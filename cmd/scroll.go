package cmd

import (
	"fmt"
	"strings"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// ScrollResult is the YAML output of a successful scroll.
type ScrollResult struct {
	OK        bool   `yaml:"ok"`
	Action    string `yaml:"action"`
	Direction string `yaml:"direction"`
	Amount    int    `yaml:"amount"`
	X         int    `yaml:"x"`
	Y         int    `yaml:"y"`
}

var scrollCmd = &cobra.Command{
	Use:   "scroll",
	Short: "Scroll within a window or element",
	Long:  "Scroll up, down, left, or right within a window or specific element.",
	RunE:  runScroll,
}

func init() {
	rootCmd.AddCommand(scrollCmd)
	scrollCmd.Flags().String("direction", "", "Scroll direction: up, down, left, right")
	scrollCmd.Flags().Int("amount", 3, "Number of scroll clicks")
	scrollCmd.Flags().Int("x", 0, "Scroll at specific X coordinate")
	scrollCmd.Flags().Int("y", 0, "Scroll at specific Y coordinate")
	scrollCmd.Flags().Int("id", 0, "Scroll within element by ID")
	scrollCmd.Flags().String("app", "", "Scope to application")
	scrollCmd.Flags().String("window", "", "Scope to window")
}

func runScroll(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.Inputter == nil {
		return fmt.Errorf("input simulation not available on this platform")
	}

	direction, _ := cmd.Flags().GetString("direction")
	amount, _ := cmd.Flags().GetInt("amount")
	x, _ := cmd.Flags().GetInt("x")
	y, _ := cmd.Flags().GetInt("y")
	id, _ := cmd.Flags().GetInt("id")
	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")

	if direction == "" {
		return fmt.Errorf("--direction is required (up, down, left, right)")
	}

	// Validate direction and compute dx/dy
	var dx, dy int
	switch strings.ToLower(direction) {
	case "up":
		dy = amount
	case "down":
		dy = -amount
	case "left":
		dx = amount
	case "right":
		dx = -amount
	default:
		return fmt.Errorf("invalid direction %q: use up, down, left, or right", direction)
	}

	// If --id specified, resolve element center coordinates
	if id > 0 {
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
			return fmt.Errorf("element with ID %d not found", id)
		}
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	}

	if err := provider.Inputter.Scroll(x, y, dx, dy); err != nil {
		return err
	}

	return output.PrintYAML(ScrollResult{
		OK:        true,
		Action:    "scroll",
		Direction: strings.ToLower(direction),
		Amount:    amount,
		X:         x,
		Y:         y,
	})
}
