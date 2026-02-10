package cmd

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
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
	readCmd.Flags().Int("max-elements", 0, "Max elements in agent format output (0 = unlimited; auto-set to 200 for web content)")
	readCmd.Flags().Int64("since", 0, "Return only changes since this timestamp (from a previous read's ts field)")

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
	maxElements, _ := cmd.Flags().GetInt("max-elements")
	since, _ := cmd.Flags().GetInt64("since")

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

		// Auto max-elements for agent format on web content (unless explicitly set)
		if output.OutputFormat == output.FormatAgent && hasWeb && !cmd.Flags().Changed("max-elements") && maxElements == 0 {
			maxElements = 200
			smartDefaults = append(smartDefaults, "max-elements=200 (web content)")
		}
	}

	smartDefaultsStr := strings.Join(smartDefaults, ", ")

	// Set max elements for agent format output
	output.MaxAgentElements = maxElements

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

	// Screenshot format: combined visual + structured output
	if output.OutputFormat == output.FormatScreenshot {
		return runReadScreenshot(cmd, provider, appName, window, windowID, pid, windowTitle, elements, prune)
	}

	// Apply pruning before flattening (applies to all output paths)
	if prune {
		elements = model.PruneEmptyGroups(elements)
	}

	// Generate stable refs before flattening so they appear in YAML/JSON output
	model.GenerateRefs(elements)

	// Flatten elements (needed for --since diff and --flat output)
	flatElements := model.FlattenElements(elements)
	if flat && text != "" {
		flatElements = model.FilterFlatByText(flatElements, text)
	}
	if flat && prune {
		flatElements = model.PruneEmptyGroupsFlat(flatElements)
	}

	now := time.Now().Unix()

	// Clean old snapshots in the background
	go model.CleanSnapshots(appName, 60*time.Second)

	// Diff mode: return only changes since the given timestamp
	if since > 0 {
		prevElements, err := model.LoadSnapshot(appName, since)
		if err != nil {
			return fmt.Errorf("no snapshot found for ts %d: %w", since, err)
		}
		diff := model.DiffElementsByHash(prevElements, flatElements)

		// Save current snapshot for future diffs
		model.SaveSnapshot(appName, now, flatElements)

		result := output.ReadDiffResult{
			App:           appName,
			PID:           pid,
			Window:        windowTitle,
			SmartDefaults: smartDefaultsStr,
			TS:            now,
			Since:         since,
			Diff:          diff,
		}
		return output.Print(result)
	}

	// Save snapshot for future --since calls
	model.SaveSnapshot(appName, now, flatElements)

	// Output as flat list or tree
	if flat {
		result := output.ReadFlatResult{
			App:           appName,
			PID:           pid,
			Window:        windowTitle,
			SmartDefaults: smartDefaultsStr,
			TS:            now,
			Elements:      flatElements,
		}
		return output.Print(result)
	}

	result := output.ReadResult{
		App:           appName,
		PID:           pid,
		Window:        windowTitle,
		SmartDefaults: smartDefaultsStr,
		TS:            now,
		Elements:      elements,
	}

	return output.Print(result)
}

// runReadScreenshot implements the --format screenshot mode: captures an annotated
// screenshot with [id] labels and returns it alongside a structured element list.
func runReadScreenshot(cmd *cobra.Command, provider *platform.Provider, appName, window string, windowID, pid int, windowTitle string, elements []model.Element, prune bool) error {
	if provider.Screenshotter == nil {
		return fmt.Errorf("screenshot not supported on this platform")
	}

	// Parse screenshot-specific flags
	scale, _ := cmd.Flags().GetFloat64("scale")
	screenshotOutput, _ := cmd.Flags().GetString("screenshot-output")
	imgFormat, _ := cmd.Flags().GetString("image-format")
	quality, _ := cmd.Flags().GetInt("quality")
	allElements, _ := cmd.Flags().GetBool("all-elements")

	// Resolve the target window for screenshot capture and bounds mapping
	listOpts := platform.ListOptions{}
	if appName != "" {
		listOpts.App = appName
	}
	if pid != 0 {
		listOpts.PID = pid
	}

	allWindows, err := provider.Reader.ListWindows(listOpts)
	if err != nil {
		return fmt.Errorf("failed to list windows: %w", err)
	}
	if len(allWindows) == 0 {
		return fmt.Errorf("no windows available")
	}

	var targetWindow model.Window
	if windowID != 0 {
		found := false
		for _, w := range allWindows {
			if w.ID == windowID {
				targetWindow = w
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("window ID %d not found", windowID)
		}
	} else if window != "" {
		found := false
		for _, w := range allWindows {
			if strings.Contains(strings.ToLower(w.Title), strings.ToLower(window)) {
				targetWindow = w
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no window found matching title %q", window)
		}
	} else {
		targetWindow = allWindows[0]
	}

	// Filter elements for annotation: default to interactive only unless --all-elements
	var annotationElements []model.Element
	if allElements {
		annotationElements = elements
	} else {
		interactiveRoles := model.ExpandRoles([]string{"interactive"})
		annotationElements = model.FilterElements(elements, interactiveRoles, nil)
	}

	if prune {
		annotationElements = model.PruneEmptyGroups(annotationElements)
	}

	// Flatten for annotation
	flatAnnotation := flattenElementsForAnnotation(annotationElements)

	// Filter out zero-bound elements (off-screen/virtualized)
	var visibleAnnotation []model.Element
	for _, el := range flatAnnotation {
		if el.Bounds[2] > 0 && el.Bounds[3] > 0 {
			visibleAnnotation = append(visibleAnnotation, el)
		}
	}

	// Capture screenshot
	screenshotOpts := platform.ScreenshotOptions{
		WindowID: targetWindow.ID,
		Format:   imgFormat,
		Quality:  quality,
		Scale:    scale,
	}

	imageData, err := provider.Screenshotter.CaptureWindow(screenshotOpts)
	if err != nil {
		return fmt.Errorf("failed to capture screenshot: %w", err)
	}

	// Decode image
	var img image.Image
	switch imgFormat {
	case "jpg", "jpeg":
		img, err = jpeg.Decode(bytes.NewReader(imageData))
	default:
		img, err = png.Decode(bytes.NewReader(imageData))
	}
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Annotate with [id] labels
	annotatedImg, err := AnnotateScreenshotWithMode(img, visibleAnnotation, targetWindow.Bounds, LabelIDs)
	if err != nil {
		return fmt.Errorf("failed to annotate screenshot: %w", err)
	}

	// Encode annotated image
	buf := &bytes.Buffer{}
	switch imgFormat {
	case "jpg", "jpeg":
		err = jpeg.Encode(buf, annotatedImg, &jpeg.Options{Quality: quality})
	default:
		err = png.Encode(buf, annotatedImg)
	}
	if err != nil {
		return fmt.Errorf("failed to encode annotated image: %w", err)
	}
	outputData := buf.Bytes()

	// Generate agent-format element list
	agentStr := output.FormatAgentString(appName, targetWindow.PID, windowTitle, elements)

	// If --screenshot-output specified, save image to file
	if screenshotOutput != "" {
		if err := os.WriteFile(screenshotOutput, outputData, 0644); err != nil {
			return fmt.Errorf("failed to write screenshot: %w", err)
		}
	}

	// Build result
	imageStr := ""
	if screenshotOutput == "" {
		imageStr = base64.StdEncoding.EncodeToString(outputData)
	} else {
		imageStr = screenshotOutput
	}

	result := output.ScreenshotReadResult{
		OK:       true,
		Action:   "read",
		App:      appName,
		PID:      targetWindow.PID,
		Window:   windowTitle,
		Image:    imageStr,
		Elements: agentStr,
	}

	return output.PrintYAML(result)
}
