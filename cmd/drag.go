package cmd

import (
	"fmt"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// DragResult is the YAML output of a successful drag.
type DragResult struct {
	OK     bool   `yaml:"ok"     json:"ok"`
	Action string `yaml:"action" json:"action"`
	FromX  int    `yaml:"from_x" json:"from_x"`
	FromY  int    `yaml:"from_y" json:"from_y"`
	ToX    int    `yaml:"to_x"   json:"to_x"`
	ToY    int    `yaml:"to_y"   json:"to_y"`
}

var dragCmd = &cobra.Command{
	Use:   "drag",
	Short: "Drag from one point to another",
	Long:  "Drag from one point to another using coordinates or element IDs.",
	RunE:  runDrag,
}

func init() {
	rootCmd.AddCommand(dragCmd)
	dragCmd.Flags().Int("from-x", 0, "Start X coordinate")
	dragCmd.Flags().Int("from-y", 0, "Start Y coordinate")
	dragCmd.Flags().Int("to-x", 0, "End X coordinate")
	dragCmd.Flags().Int("to-y", 0, "End Y coordinate")
	dragCmd.Flags().Int("from-id", 0, "Start element (center)")
	dragCmd.Flags().Int("to-id", 0, "End element (center)")
	dragCmd.Flags().String("app", "", "Scope to application")
	dragCmd.Flags().String("window", "", "Scope to window")
}

func runDrag(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.Inputter == nil {
		return fmt.Errorf("input simulation not available on this platform")
	}

	fromX, _ := cmd.Flags().GetInt("from-x")
	fromY, _ := cmd.Flags().GetInt("from-y")
	toX, _ := cmd.Flags().GetInt("to-x")
	toY, _ := cmd.Flags().GetInt("to-y")
	fromID, _ := cmd.Flags().GetInt("from-id")
	toID, _ := cmd.Flags().GetInt("to-id")
	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")

	hasCoords := cmd.Flags().Changed("from-x") || cmd.Flags().Changed("from-y") ||
		cmd.Flags().Changed("to-x") || cmd.Flags().Changed("to-y")
	hasIDs := cmd.Flags().Changed("from-id") || cmd.Flags().Changed("to-id")

	if !hasCoords && !hasIDs {
		return fmt.Errorf("specify --from-x/--from-y and --to-x/--to-y or --from-id/--to-id")
	}

	// Resolve element IDs to coordinates if specified
	if hasIDs {
		if appName == "" && window == "" {
			return fmt.Errorf("--from-id/--to-id requires --app or --window to scope the element lookup")
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

		if fromID > 0 {
			elem := findElementByID(elements, fromID)
			if elem == nil {
				return fmt.Errorf("from-element with ID %d not found", fromID)
			}
			fromX = elem.Bounds[0] + elem.Bounds[2]/2
			fromY = elem.Bounds[1] + elem.Bounds[3]/2
		}

		if toID > 0 {
			elem := findElementByID(elements, toID)
			if elem == nil {
				return fmt.Errorf("to-element with ID %d not found", toID)
			}
			toX = elem.Bounds[0] + elem.Bounds[2]/2
			toY = elem.Bounds[1] + elem.Bounds[3]/2
		}
	}

	if err := provider.Inputter.Drag(fromX, fromY, toX, toY); err != nil {
		return err
	}

	return output.Print(DragResult{
		OK:     true,
		Action: "drag",
		FromX:  fromX,
		FromY:  fromY,
		ToX:    toX,
		ToY:    toY,
	})
}
