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

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

var screenshotCoordsCmd = &cobra.Command{
	Use:   "screenshot-coords",
	Short: "Capture a screenshot with coordinate labels on UI elements",
	Long:  "Capture a screenshot and annotate it with bounding boxes and coordinate labels for UI elements, making it easy to see where clickable/interactive elements are located.",
	RunE:  runScreenshotCoords,
}

func init() {
	rootCmd.AddCommand(screenshotCoordsCmd)

	// Screenshot-specific flags (same as screenshot command)
	screenshotCoordsCmd.Flags().String("window", "", "Capture window by title substring")
	screenshotCoordsCmd.Flags().String("app", "", "Capture specific app's frontmost window")
	screenshotCoordsCmd.Flags().Int("window-id", 0, "Capture window by system ID")
	screenshotCoordsCmd.Flags().Int("pid", 0, "Capture frontmost window of this PID")
	screenshotCoordsCmd.Flags().String("output", "", "Output file path (default: stdout as base64)")
	screenshotCoordsCmd.Flags().String("format", "png", "Output format: png, jpg")
	screenshotCoordsCmd.Flags().Int("quality", 80, "JPEG quality 1-100")
	screenshotCoordsCmd.Flags().Float64("scale", 0.5, "Scale factor 0.1-1.0 (for token efficiency)")

	// Element filtering flags (like read command)
	screenshotCoordsCmd.Flags().Bool("all-elements", false, "Label all elements (default: interactive elements only)")
	screenshotCoordsCmd.Flags().String("roles", "", "Comma-separated roles to include (e.g. \"btn,txt,lnk\")")
	screenshotCoordsCmd.Flags().Int("depth", 0, "Max depth to traverse (0 = unlimited)")
	screenshotCoordsCmd.Flags().String("text", "", "Filter elements by text content (case-insensitive substring match)")
	screenshotCoordsCmd.Flags().Bool("prune", false, "Remove anonymous group/other elements with no title/value/description")
	screenshotCoordsCmd.Flags().Bool("include-menubar", false, "Include macOS menu bar in app screenshots")
}

// flattenElementsForAnnotation converts a tree of elements into a flat list for annotation
func flattenElementsForAnnotation(elements []model.Element) []model.Element {
	var result []model.Element
	for _, el := range elements {
		flattenRecursiveForAnnotation(el, &result)
	}
	return result
}

func flattenRecursiveForAnnotation(el model.Element, result *[]model.Element) {
	*result = append(*result, el)
	for _, child := range el.Children {
		flattenRecursiveForAnnotation(child, result)
	}
}

func runScreenshotCoords(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.Screenshotter == nil {
		return fmt.Errorf("screenshot not supported on this platform")
	}
	if provider.Reader == nil {
		return fmt.Errorf("element reading not supported on this platform")
	}

	// Parse screenshot flags
	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")
	windowID, _ := cmd.Flags().GetInt("window-id")
	pid, _ := cmd.Flags().GetInt("pid")
	output, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")
	quality, _ := cmd.Flags().GetInt("quality")
	scale, _ := cmd.Flags().GetFloat64("scale")

	// Parse element filtering flags
	allElements, _ := cmd.Flags().GetBool("all-elements")
	rolesStr, _ := cmd.Flags().GetString("roles")
	depth, _ := cmd.Flags().GetInt("depth")
	text, _ := cmd.Flags().GetString("text")
	prune, _ := cmd.Flags().GetBool("prune")

	// Build roles list
	var roles []string
	if rolesStr != "" {
		for _, r := range strings.Split(rolesStr, ",") {
			roles = append(roles, strings.TrimSpace(r))
		}
		roles = model.ExpandRoles(roles)
	} else if !allElements {
		// Default to interactive elements if not specified and not --all-elements
		roles = model.ExpandRoles([]string{"interactive"})
	}

	// Resolve target window so we can get its bounds for coordinate mapping
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
		// Use the first window (frontmost)
		targetWindow = allWindows[0]
	}

	appName = targetWindow.App
	pid = targetWindow.PID
	windowID = targetWindow.ID

	// Read UI elements
	readOpts := platform.ReadOptions{
		App:         appName,
		Window:      window,
		WindowID:    windowID,
		PID:         pid,
		Depth:       depth,
		Roles:       roles,
		VisibleOnly: true,
	}

	elements, err := provider.Reader.ReadElements(readOpts)
	if err != nil {
		return err
	}

	// Apply text filter if specified
	if text != "" {
		elements = model.FilterByText(elements, text)
	}

	// Prune empty groups if requested
	if prune {
		elements = model.PruneEmptyGroups(elements)
	}

	// Flatten elements for easier iteration and annotation
	flatElements := flattenElementsForAnnotation(elements)

	includeMenuBar, _ := cmd.Flags().GetBool("include-menubar")

	// Capture screenshot of the specific target window
	screenshotOpts := platform.ScreenshotOptions{
		WindowID:       windowID,
		Format:         format,
		Quality:        quality,
		Scale:          scale,
		IncludeMenuBar: includeMenuBar,
	}

	imageData, err := provider.Screenshotter.CaptureWindow(screenshotOpts)
	if err != nil {
		return fmt.Errorf("failed to capture screenshot: %w", err)
	}

	// Decode image
	var img image.Image
	switch format {
	case "jpg", "jpeg":
		img, err = jpeg.Decode(bytes.NewReader(imageData))
	default: // png
		img, err = png.Decode(bytes.NewReader(imageData))
	}
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Annotate image with coordinate labels
	// Pass window bounds so we can convert screen-absolute element coords
	// to window-relative image coords
	annotatedImg, err := AnnotateScreenshot(img, flatElements, targetWindow.Bounds)
	if err != nil {
		return fmt.Errorf("failed to annotate screenshot: %w", err)
	}

	// Encode annotated image
	var outputData []byte
	buf := &bytes.Buffer{}
	switch format {
	case "jpg", "jpeg":
		err = jpeg.Encode(buf, annotatedImg, &jpeg.Options{Quality: quality})
	default: // png
		err = png.Encode(buf, annotatedImg)
	}
	if err != nil {
		return fmt.Errorf("failed to encode annotated image: %w", err)
	}
	outputData = buf.Bytes()

	// Output to file or stdout
	if output != "" {
		return os.WriteFile(output, outputData, 0644)
	}

	// Default: write to stdout as base64 for easy agent consumption
	encoder := base64.NewEncoder(base64.StdEncoding, os.Stdout)
	if _, err := encoder.Write(outputData); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	fmt.Println() // newline after base64
	return nil
}
