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
	readCmd.Flags().Bool("prune", false, "Remove anonymous group/other elements that have no title, value, or description")
	readCmd.Flags().Bool("focused", false, "Only return the currently focused element")
	readCmd.Flags().Int("scope-id", 0, "Limit to descendants of this element ID")
	readCmd.Flags().Bool("children", false, "Show only direct children of the matched element (use with --text or --scope-id)")

	// Screenshot format flags (only used with --format screenshot)
	readCmd.Flags().Float64("scale", 0.25, "Screenshot scale factor 0.1-1.0 (default 0.25 for token efficiency, only with --format screenshot)")
	readCmd.Flags().String("screenshot-output", "", "Save screenshot to file instead of inline base64 (only with --format screenshot)")
	readCmd.Flags().String("image-format", "jpg", "Screenshot image format: png, jpg (only with --format screenshot)")
	readCmd.Flags().Int("quality", 80, "JPEG quality 1-100 (only with --format screenshot)")
	readCmd.Flags().Bool("all-elements", false, "Label all elements in screenshot (default: interactive only, only with --format screenshot)")
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
	prune, _ := cmd.Flags().GetBool("prune")
	focused, _ := cmd.Flags().GetBool("focused")
	scopeID, _ := cmd.Flags().GetInt("scope-id")
	children, _ := cmd.Flags().GetBool("children")

	var roles []string
	if rolesStr != "" {
		for _, r := range strings.Split(rolesStr, ",") {
			roles = append(roles, strings.TrimSpace(r))
		}
		roles = model.ExpandRoles(roles)
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

	// --- Smart defaults ---
	// Detect web content and apply optimal defaults unless --raw is set.
	var smartDefaults []string
	pruneExplicit := cmd.Flags().Changed("prune")

	if !output.RawMode {
		hasWeb := model.HasWebContent(elements)

		// Auto-prune for web content (unless --prune was explicitly set)
		if hasWeb && !pruneExplicit {
			prune = true
			smartDefaults = append(smartDefaults, "auto-pruned (web content detected)")
		}

		// Auto-expand roles for web content: add "other" when "input" is specified
		if rolesStr != "" {
			var expanded bool
			roles, expanded = model.ExpandRolesForWeb(roles, hasWeb)
			if expanded {
				smartDefaults = append(smartDefaults, "auto-expanded roles: added \"other\" (web input compatibility)")
			}
		}

		// Auto agent format for piped output
		if output.OutputFormat == output.FormatAgent && !cmd.Root().PersistentFlags().Changed("format") {
			smartDefaults = append(smartDefaults, "agent format (piped output)")
		}
	}

	smartDefaultsStr := strings.Join(smartDefaults, ", ")

	// Scope to descendants of a specific element
	if scopeID > 0 {
		scopeEl := findElementByID(elements, scopeID)
		if scopeEl == nil {
			return fmt.Errorf("scope element with id %d not found", scopeID)
		}
		if children {
			// --children with --scope-id: direct children only (strip grandchildren)
			elements = directChildrenOnly(scopeEl.Children)
		} else {
			elements = scopeEl.Children
		}
	}

	// Apply text filter
	if text != "" {
		if children && scopeID == 0 {
			// --children with --text: find the matching element and return its direct children
			matched := model.FindFirstByText(elements, text)
			if matched == nil {
				return fmt.Errorf("no element found matching text %q", text)
			}
			elements = directChildrenOnly(matched.Children)
		} else {
			elements = model.FilterByText(elements, text)
		}
	}

	// Apply focused filter
	if focused {
		elements = model.FilterByFocused(elements)
	}

	// Resolve window title from the element tree
	windowTitle := window
	if windowTitle == "" {
		for _, el := range elements {
			if el.Role == "window" && el.Title != "" {
				windowTitle = el.Title
				break
			}
		}
	}

	// Output as flat list or tree
	if flat {
		flatElements := model.FlattenElements(elements)
		if text != "" {
			flatElements = model.FilterFlatByText(flatElements, text)
		}
		if prune {
			flatElements = model.PruneEmptyGroupsFlat(flatElements)
		}
		result := output.ReadFlatResult{
			App:           appName,
			PID:           pid,
			Window:        windowTitle,
			SmartDefaults: smartDefaultsStr,
			TS:            time.Now().Unix(),
			Elements:      flatElements,
		}
		return output.Print(result)
	}

	if prune {
		elements = model.PruneEmptyGroups(elements)
	}

	result := output.ReadResult{
		App:           appName,
		PID:           pid,
		Window:        windowTitle,
		SmartDefaults: smartDefaultsStr,
		TS:            time.Now().Unix(),
		Elements:      elements,
	}

	return output.Print(result)
}
