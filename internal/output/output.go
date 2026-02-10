package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mj1618/desktop-cli/internal/model"
	"gopkg.in/yaml.v3"
)

// yamlBoolKeyRe matches YAML 1.1 boolean keywords used as mapping keys that
// gopkg.in/yaml.v3 unnecessarily quotes (e.g. "y":, "n":, "yes":, "no":).
var yamlBoolKeyRe = regexp.MustCompile(`(?m)^(\s*)"(y|Y|n|N|yes|Yes|YES|no|No|NO|on|On|ON|off|Off|OFF)":`)

// Format represents the output format.
type Format string

const (
	FormatYAML       Format = "yaml"
	FormatJSON       Format = "json"
	FormatAgent      Format = "agent"
	FormatScreenshot Format = "screenshot"
)

// OutputFormat is the current output format, set by the root command's --format flag.
var OutputFormat Format = FormatYAML

// PrettyOutput enables pretty-printing for JSON output.
var PrettyOutput bool

// RawMode disables all smart defaults when true (set by --raw flag).
var RawMode bool

// MaxAgentElements limits how many elements agent format will output.
// 0 means unlimited. Set via --max-elements or auto-applied for web content.
var MaxAgentElements int

// IsOutputPiped returns true when stdout is a pipe (not a terminal).
// When an agent calls the CLI, stdout is typically piped.
func IsOutputPiped() bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// ReadResult is the top-level output of the `read` command.
type ReadResult struct {
	App           string          `yaml:"app,omitempty"            json:"app,omitempty"`
	PID           int             `yaml:"pid,omitempty"            json:"pid,omitempty"`
	Window        string          `yaml:"window,omitempty"         json:"window,omitempty"`
	SmartDefaults string          `yaml:"smart_defaults,omitempty" json:"smart_defaults,omitempty"`
	TS            int64           `yaml:"ts"                       json:"ts"`
	Elements      []model.Element `yaml:"elements"                 json:"elements"`
}

// ReadFlatResult is the top-level output when --flat is used.
type ReadFlatResult struct {
	App           string              `yaml:"app,omitempty"            json:"app,omitempty"`
	PID           int                 `yaml:"pid,omitempty"            json:"pid,omitempty"`
	Window        string              `yaml:"window,omitempty"         json:"window,omitempty"`
	SmartDefaults string              `yaml:"smart_defaults,omitempty" json:"smart_defaults,omitempty"`
	TS            int64               `yaml:"ts"                       json:"ts"`
	Elements      []model.FlatElement `yaml:"elements"                 json:"elements"`
}

// ReadDiffResult is the output when --since is used, returning only changes.
type ReadDiffResult struct {
	App           string         `yaml:"app,omitempty"            json:"app,omitempty"`
	PID           int            `yaml:"pid,omitempty"            json:"pid,omitempty"`
	Window        string         `yaml:"window,omitempty"         json:"window,omitempty"`
	SmartDefaults string         `yaml:"smart_defaults,omitempty" json:"smart_defaults,omitempty"`
	TS            int64          `yaml:"ts"                       json:"ts"`
	Since         int64          `yaml:"since"                    json:"since"`
	Diff          model.TreeDiff `yaml:"diff"                     json:"diff"`
}

// ScreenshotReadResult is the output of `read --format screenshot`, combining
// an annotated screenshot (with [id] labels) and a structured element list.
type ScreenshotReadResult struct {
	OK       bool   `yaml:"ok"               json:"ok"`
	Action   string `yaml:"action"           json:"action"`
	App      string `yaml:"app,omitempty"    json:"app,omitempty"`
	PID      int    `yaml:"pid,omitempty"    json:"pid,omitempty"`
	Window   string `yaml:"window,omitempty" json:"window,omitempty"`
	Image    string `yaml:"image"            json:"image"`
	Elements string `yaml:"elements"         json:"elements"`
}

// Print serializes v to stdout in the current output format.
func Print(v interface{}) error {
	switch OutputFormat {
	case FormatJSON:
		if PrettyOutput {
			return PrintPrettyJSON(v)
		}
		return PrintJSON(v)
	case FormatYAML:
		return PrintYAML(v)
	case FormatAgent:
		return PrintAgent(v)
	case FormatScreenshot:
		// Screenshot format is handled directly in cmd/read.go's runReadScreenshot;
		// if we reach here from a non-read command, fall back to YAML.
		return PrintYAML(v)
	default:
		return fmt.Errorf("unsupported output format: %s", OutputFormat)
	}
}

// PrintJSON serializes v to stdout as compact single-line JSON.
func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// PrintPrettyJSON serializes v to stdout as indented JSON.
func PrintPrettyJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// PrintYAML serializes v to stdout as YAML.
// It post-processes the output to unquote YAML 1.1 boolean keywords (e.g. "y")
// that gopkg.in/yaml.v3 defensively quotes when used as mapping keys.
func PrintYAML(v interface{}) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("yaml encode: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("yaml close: %w", err)
	}
	out := yamlBoolKeyRe.ReplaceAll(buf.Bytes(), []byte("${1}${2}:"))
	_, err := os.Stdout.Write(out)
	return err
}

// PrintAgent renders a compact, AI-agent-friendly text format showing only
// interactive (pressable) elements, one per line.
func PrintAgent(v interface{}) error {
	switch result := v.(type) {
	case ReadResult:
		return printAgentTree(result.App, result.PID, result.Window, result.Elements)
	case ReadFlatResult:
		return printAgentFlat(result.App, result.PID, result.Window, result.Elements)
	case ReadDiffResult:
		return printAgentDiff(result)
	default:
		return PrintYAML(v)
	}
}

func printAgentTree(app string, pid int, window string, elements []model.Element) error {
	model.GenerateRefs(elements)
	flat := model.FlattenElements(elements)
	return printAgentFlat(app, pid, window, flat)
}

func printAgentFlat(app string, pid int, window string, elements []model.FlatElement) error {
	s := formatAgentString(app, pid, window, elements)
	_, err := os.Stdout.WriteString(s)
	return err
}

// FormatAgentString renders an element tree as a compact agent-format string.
// This is the same format as --format agent, but returned as a string for
// embedding in other responses (e.g. the --post-read state field).
func FormatAgentString(app string, pid int, window string, elements []model.Element) string {
	model.GenerateRefs(elements)
	flat := model.FlattenElements(elements)
	return formatAgentString(app, pid, window, flat)
}

// FormatAgentStringWithMax is like FormatAgentString but accepts an explicit
// max-elements cap (0 = unlimited). Used by --post-read to apply sensible
// defaults without relying on the global MaxAgentElements variable.
func FormatAgentStringWithMax(app string, pid int, window string, elements []model.Element, maxElements int) string {
	model.GenerateRefs(elements)
	flat := model.FlattenElements(elements)
	return formatAgentStringWithMax(app, pid, window, flat, maxElements)
}

func formatAgentString(app string, pid int, window string, elements []model.FlatElement) string {
	return formatAgentStringWithMax(app, pid, window, elements, MaxAgentElements)
}

func formatAgentStringWithMax(app string, pid int, window string, elements []model.FlatElement, maxEl int) string {
	var buf bytes.Buffer

	// Header
	header := agentHeader(app, pid, window, elements)
	if header != "" {
		fmt.Fprintf(&buf, "# %s\n\n", header)
	}

	// Filter to interactive elements and display text, then format.
	// Skip elements with zero-width or zero-height bounds — these are
	// off-screen or virtualized and not actually visible to the user.
	count := 0
	totalVisible := 0
	for _, el := range elements {
		if el.Bounds[2] <= 0 || el.Bounds[3] <= 0 {
			continue
		}
		isInteractive := hasAction(el.Actions, "press")
		isDisplayText := el.Role == "txt" && el.Value != ""
		if !isInteractive && !isDisplayText {
			continue
		}
		totalVisible++
		if maxEl > 0 && count >= maxEl {
			continue // keep counting totalVisible for the truncation message
		}
		fmt.Fprintln(&buf, formatAgentLine(el))
		count++
	}

	if maxEl > 0 && totalVisible > maxEl {
		fmt.Fprintf(&buf, "\n# ... truncated: showing %d of %d elements. Use --max-elements 0 for all, or --roles/--depth/--text to filter.\n", maxEl, totalVisible)
	}

	return buf.String()
}

func agentHeader(app string, pid int, window string, elements []model.FlatElement) string {
	// Try to get window title from the first window-role element
	winTitle := window
	if winTitle == "" {
		for _, el := range elements {
			if el.Role == "window" && el.Title != "" {
				winTitle = el.Title
				break
			}
		}
	}

	var parts []string
	if winTitle != "" {
		parts = append(parts, winTitle)
	}
	if app != "" && app != winTitle {
		parts = append(parts, app)
	}
	header := strings.Join(parts, " - ")
	if pid != 0 {
		if header != "" {
			header = fmt.Sprintf("%s (pid: %d)", header, pid)
		} else {
			header = fmt.Sprintf("pid: %d", pid)
		}
	}
	return header
}

func formatAgentLine(el model.FlatElement) string {
	label := el.Title
	if label == "" {
		label = el.Description
	}
	// For display text elements, prefer value as label if title/description are empty
	isDisplayText := el.Role == "txt" && el.Value != "" && !hasAction(el.Actions, "press")
	if isDisplayText && label == "" {
		label = el.Value
	}
	label = truncate(label, 80)

	var idPart string
	if el.Ref != "" {
		idPart = fmt.Sprintf("[%d|%s]", el.ID, el.Ref)
	} else {
		idPart = fmt.Sprintf("[%d]", el.ID)
	}

	line := fmt.Sprintf("%s %s %q (%d,%d,%d,%d)",
		idPart, el.Role, label,
		el.Bounds[0], el.Bounds[1], el.Bounds[2], el.Bounds[3])

	// Annotations
	if isDisplayText {
		line += " display"
	}
	if el.Enabled != nil && !*el.Enabled {
		line += " disabled"
	}
	if el.Selected {
		line += " selected"
	}
	if el.Focused {
		line += " focused"
	}
	if (el.Role == "chk" || el.Role == "toggle") && el.Value != "" {
		if el.Value == "1" {
			line += " checked"
		} else {
			line += " unchecked"
		}
	}
	if el.Value != "" && el.Role != "chk" && el.Role != "toggle" {
		// For display text, show val if label came from title/desc (not value itself)
		if !isDisplayText || (el.Title != "" || el.Description != "") {
			line += fmt.Sprintf(" val=%q", truncate(el.Value, 60))
		}
	}

	return line
}

func hasAction(actions []string, target string) bool {
	for _, a := range actions {
		if a == target {
			return true
		}
	}
	return false
}

func printAgentDiff(result ReadDiffResult) error {
	var buf bytes.Buffer

	// Header with diff annotation
	header := agentHeader(result.App, result.PID, result.Window, nil)
	if header != "" {
		fmt.Fprintf(&buf, "# %s [diff since %d]\n\n", header, result.Since)
	} else {
		fmt.Fprintf(&buf, "# diff since %d\n\n", result.Since)
	}

	// Added elements
	for _, el := range result.Diff.Added {
		fmt.Fprintf(&buf, "+ %s\n", formatAgentLine(el))
	}

	// Removed elements
	for _, el := range result.Diff.Removed {
		label := el.Title
		if label == "" {
			label = el.Description
		}
		if label == "" {
			label = el.Value
		}
		if label != "" {
			fmt.Fprintf(&buf, "- [%d] %s %q\n", el.ID, el.Role, truncate(label, 80))
		} else {
			fmt.Fprintf(&buf, "- [%d] %s\n", el.ID, el.Role)
		}
	}

	// Changed elements
	for _, ch := range result.Diff.Changed {
		label := ch.Title
		if label == "" {
			label = ch.Role
		}
		var parts []string
		for field, vals := range ch.Changes {
			parts = append(parts, fmt.Sprintf("%s=%q (was %q)", field, vals[1], vals[0]))
		}
		fmt.Fprintf(&buf, "~ [%d] %s %s\n", ch.ID, label, strings.Join(parts, ", "))
	}

	// Unchanged count
	if result.Diff.UnchangedCount > 0 {
		fmt.Fprintf(&buf, "# %d elements unchanged\n", result.Diff.UnchangedCount)
	}

	_, err := os.Stdout.WriteString(buf.String())
	return err
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
