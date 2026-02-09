package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read",
	Short: "Read the UI element tree and output as YAML",
	Long:  "Read the UI element tree from the OS accessibility layer and output as structured YAML.",
	RunE:  runRead,
}

func init() {
	rootCmd.AddCommand(readCmd)
	readCmd.Flags().String("app", "", "Filter to a specific application by name")
	readCmd.Flags().String("window", "", "Filter to a specific window by title substring")
	readCmd.Flags().Int("window-id", 0, "Filter to a specific window by system window ID")
	readCmd.Flags().Int("pid", 0, "Filter to a specific process by PID")
	readCmd.Flags().Int("depth", 0, "Max depth to traverse (0 = unlimited)")
	readCmd.Flags().String("roles", "", "Comma-separated roles to include (e.g. \"btn,txt,lnk\")")
	readCmd.Flags().Bool("visible-only", true, "Only include visible/on-screen elements")
	readCmd.Flags().String("bbox", "", "Only include elements within bounding box (x,y,w,h)")
	readCmd.Flags().Bool("compact", false, "Ultra-compact output: flatten tree, minimal keys")
	readCmd.Flags().Bool("pretty", false, "Pretty-print output (no-op for YAML, which is always human-readable)")
	readCmd.Flags().String("text", "", "Filter elements by text content (case-insensitive substring match on title, value, description)")
	readCmd.Flags().Bool("flat", false, "Output as flat list with path breadcrumbs instead of nested tree")
}

func runRead(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}

	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")
	windowID, _ := cmd.Flags().GetInt("window-id")
	pid, _ := cmd.Flags().GetInt("pid")
	depth, _ := cmd.Flags().GetInt("depth")
	rolesStr, _ := cmd.Flags().GetString("roles")
	visibleOnly, _ := cmd.Flags().GetBool("visible-only")
	bboxStr, _ := cmd.Flags().GetString("bbox")
	compact, _ := cmd.Flags().GetBool("compact")
	text, _ := cmd.Flags().GetString("text")
	flat, _ := cmd.Flags().GetBool("flat")

	var roles []string
	if rolesStr != "" {
		for _, r := range strings.Split(rolesStr, ",") {
			roles = append(roles, strings.TrimSpace(r))
		}
	}

	var bbox *platform.Bounds
	if bboxStr != "" {
		bbox, err = platform.ParseBBox(bboxStr)
		if err != nil {
			return err
		}
	}

	if provider.Reader == nil {
		return fmt.Errorf("reader not available on this platform")
	}

	opts := platform.ReadOptions{
		App:         appName,
		Window:      window,
		WindowID:    windowID,
		PID:         pid,
		Depth:       depth,
		Roles:       roles,
		VisibleOnly: visibleOnly,
		BBox:        bbox,
		Compact:     compact,
	}

	elements, err := provider.Reader.ReadElements(opts)
	if err != nil {
		return err
	}

	// Apply text filter
	if text != "" {
		elements = model.FilterByText(elements, text)
	}

	// Output as flat list or tree
	if flat {
		flatElements := model.FlattenElements(elements)
		result := output.ReadFlatResult{
			App:      appName,
			PID:      pid,
			TS:       time.Now().Unix(),
			Elements: flatElements,
		}
		return output.Print(result)
	}

	result := output.ReadResult{
		App:      appName,
		PID:      pid,
		TS:       time.Now().Unix(),
		Elements: elements,
	}

	return output.Print(result)
}
