package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// FillResult is the YAML output of a successful fill command.
type FillResult struct {
	OK        bool              `yaml:"ok"                  json:"ok"`
	Action    string            `yaml:"action"              json:"action"`
	FieldsSet int               `yaml:"fields_set"          json:"fields_set"`
	Results   []FillFieldResult `yaml:"results"             json:"results"`
	Submitted *ElementInfo      `yaml:"submitted,omitempty" json:"submitted,omitempty"`
	State     string            `yaml:"state,omitempty"     json:"state,omitempty"`
}

// FillFieldResult is the output for a single field within a fill command.
type FillFieldResult struct {
	Label  string       `yaml:"label,omitempty" json:"label,omitempty"`
	ID     int          `yaml:"id,omitempty"    json:"id,omitempty"`
	OK     bool         `yaml:"ok"              json:"ok"`
	Error  string       `yaml:"error,omitempty" json:"error,omitempty"`
	Target *ElementInfo `yaml:"target,omitempty" json:"target,omitempty"`
}

// fillYAMLInput is the YAML structure for stdin input.
type fillYAMLInput struct {
	Fields []fillYAMLField `yaml:"fields"`
	Submit string          `yaml:"submit,omitempty"`
}

type fillYAMLField struct {
	Label  string `yaml:"label,omitempty"`
	ID     int    `yaml:"id,omitempty"`
	Value  string `yaml:"value"`
	Method string `yaml:"method,omitempty"`
}

var fillCmd = &cobra.Command{
	Use:   "fill",
	Short: "Set multiple form fields in one call",
	Long: `Fill multiple form fields in one call, reading the UI tree only once.

Use --field flags to specify label=value pairs, or pipe YAML on stdin.
All fields are resolved from a single tree read, making this much faster
than calling type --target or set-value multiple times.

Examples:
  desktop-cli fill --app "Safari" --field "Name=John" --field "Email=john@example.com"
  desktop-cli fill --app "Safari" --field "Name=John" --submit "Submit"
  desktop-cli fill --app "Chrome" --field "Search=query" --method type
  desktop-cli fill --app "Safari" --field "id:42=John Doe"`,
	RunE: runFill,
}

func init() {
	rootCmd.AddCommand(fillCmd)
	fillCmd.Flags().String("app", "", "Target application (required)")
	fillCmd.Flags().String("window", "", "Target window")
	fillCmd.Flags().Int("window-id", 0, "Target by window ID")
	fillCmd.Flags().Int("pid", 0, "Target by PID")
	fillCmd.Flags().StringArray("field", nil, `Set a field: "Label=value" or "id:42=value" (repeatable)`)
	fillCmd.Flags().String("submit", "", "After filling, click element with this text")
	fillCmd.Flags().Bool("tab-between", false, "Use Tab key to move between fields instead of direct targeting")
	fillCmd.Flags().String("method", "set-value", `How to set values: "set-value" (direct, default) or "type" (keystrokes)`)
	addPostReadFlags(fillCmd)
}

// parsedField represents a single field to fill.
type parsedField struct {
	label  string // text label to find (empty if using ID)
	id     int    // element ID (0 if using label)
	value  string
	method string // "set-value" or "type" (empty = use command default)
}

func runFill(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}

	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")
	windowID, _ := cmd.Flags().GetInt("window-id")
	pid, _ := cmd.Flags().GetInt("pid")
	submitText, _ := cmd.Flags().GetString("submit")
	tabBetween, _ := cmd.Flags().GetBool("tab-between")
	defaultMethod, _ := cmd.Flags().GetString("method")

	if err := requireScope(appName, window, windowID, pid); err != nil {
		return err
	}

	if defaultMethod != "set-value" && defaultMethod != "type" {
		return fmt.Errorf("--method must be \"set-value\" or \"type\", got %q", defaultMethod)
	}

	// Parse fields from --field flags or stdin YAML
	fields, stdinSubmit, err := parseFields(cmd, defaultMethod)
	if err != nil {
		return err
	}
	if len(fields) == 0 {
		return fmt.Errorf("no fields specified — use --field flags or pipe YAML on stdin")
	}
	if submitText == "" && stdinSubmit != "" {
		submitText = stdinSubmit
	}

	// 1. Read the tree ONCE for all fields
	if provider.Reader == nil {
		return fmt.Errorf("reader not available on this platform")
	}
	elements, err := provider.Reader.ReadElements(platform.ReadOptions{
		App:      appName,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
	})
	if err != nil {
		return fmt.Errorf("failed to read elements: %w", err)
	}

	// 2. Resolve and fill each field
	results := make([]FillFieldResult, 0, len(fields))
	fieldsSet := 0

	for i, f := range fields {
		result := fillOneField(provider, elements, f, appName, window, windowID, pid, tabBetween && i > 0)
		results = append(results, result)
		if result.OK {
			fieldsSet++
		}
	}

	// 3. Submit if requested
	var submitted *ElementInfo
	if submitText != "" {
		elem, err := resolveElementByTextFromTree(elements, submitText, "", false, 0)
		if err != nil {
			// Re-read the tree in case the form changed after filling
			elements2, readErr := provider.Reader.ReadElements(platform.ReadOptions{
				App:      appName,
				Window:   window,
				WindowID: windowID,
				PID:      pid,
			})
			if readErr == nil {
				elem, err = resolveElementByTextFromTree(elements2, submitText, "", false, 0)
			}
		}
		if err != nil {
			return fmt.Errorf("submit element %q not found: %w", submitText, err)
		}
		// Click the submit element
		if provider.Inputter != nil {
			cx := elem.Bounds[0] + elem.Bounds[2]/2
			cy := elem.Bounds[1] + elem.Bounds[3]/2
			if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
				return fmt.Errorf("failed to click submit element: %w", err)
			}
		} else if provider.ActionPerformer != nil {
			if err := provider.ActionPerformer.PerformAction(platform.ActionOptions{
				App:      appName,
				Window:   window,
				WindowID: windowID,
				PID:      pid,
				ID:       elem.ID,
				Action:   "press",
			}); err != nil {
				return fmt.Errorf("failed to press submit element: %w", err)
			}
		}
		submitted = elementInfoFromElement(elem)
	}

	// Post-read state
	prOpts := getPostReadOptions(cmd)
	var state string
	if prOpts.PostRead {
		state = readPostActionState(provider, appName, window, windowID, pid, prOpts.Delay, prOpts.MaxElements)
	}

	return output.Print(FillResult{
		OK:        fieldsSet == len(fields),
		Action:    "fill",
		FieldsSet: fieldsSet,
		Results:   results,
		Submitted: submitted,
		State:     state,
	})
}

// fillOneField resolves and fills a single form field.
func fillOneField(provider *platform.Provider, elements []model.Element, f parsedField, appName, window string, windowID, pid int, useTab bool) FillFieldResult {
	method := f.method

	// Resolve element
	var elem *model.Element
	var err error

	if f.id > 0 {
		elem = findElementByID(elements, f.id)
		if elem == nil {
			return FillFieldResult{
				Label: f.label,
				ID:    f.id,
				OK:    false,
				Error: fmt.Sprintf("element with id %d not found", f.id),
			}
		}
	} else {
		elem, err = resolveElementByTextFromTree(elements, f.label, "", false, 0)
		if err != nil {
			return FillFieldResult{
				Label: f.label,
				OK:    false,
				Error: err.Error(),
			}
		}
	}

	label := f.label
	if label == "" {
		label = elem.Title
	}

	// Fill the field
	if useTab {
		// Tab to next field
		if provider.Inputter != nil {
			if err := provider.Inputter.KeyCombo([]string{"tab"}); err != nil {
				return FillFieldResult{Label: label, ID: elem.ID, OK: false, Error: fmt.Sprintf("tab failed: %v", err)}
			}
			time.Sleep(50 * time.Millisecond)
		}
	}

	if method == "type" {
		// Focus element by clicking, then type
		if provider.Inputter == nil {
			return FillFieldResult{Label: label, ID: elem.ID, OK: false, Error: "input simulation not available"}
		}
		if !useTab {
			cx := elem.Bounds[0] + elem.Bounds[2]/2
			cy := elem.Bounds[1] + elem.Bounds[3]/2
			if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
				return FillFieldResult{Label: label, ID: elem.ID, OK: false, Error: fmt.Sprintf("failed to focus: %v", err)}
			}
			time.Sleep(50 * time.Millisecond)
		}
		// Select all first to replace existing content
		if err := provider.Inputter.KeyCombo([]string{"cmd", "a"}); err != nil {
			return FillFieldResult{Label: label, ID: elem.ID, OK: false, Error: fmt.Sprintf("select all failed: %v", err)}
		}
		time.Sleep(30 * time.Millisecond)
		if err := provider.Inputter.TypeText(f.value, 0); err != nil {
			return FillFieldResult{Label: label, ID: elem.ID, OK: false, Error: fmt.Sprintf("type failed: %v", err)}
		}
		time.Sleep(50 * time.Millisecond)
	} else {
		// set-value method (default): direct value setting via accessibility API
		if provider.ValueSetter == nil {
			return FillFieldResult{Label: label, ID: elem.ID, OK: false, Error: "set-value not supported on this platform"}
		}
		opts := platform.SetValueOptions{
			App:       appName,
			Window:    window,
			WindowID:  windowID,
			PID:       pid,
			ID:        elem.ID,
			Value:     f.value,
			Attribute: "value",
		}
		if err := provider.ValueSetter.SetValue(opts); err != nil {
			return FillFieldResult{Label: label, ID: elem.ID, OK: false, Error: fmt.Sprintf("set-value failed: %v", err)}
		}
	}

	return FillFieldResult{
		Label:  label,
		ID:     elem.ID,
		OK:     true,
		Target: elementInfoFromElement(elem),
	}
}

// parseFields extracts fields from --field flags or stdin YAML.
// Returns (fields, stdinSubmitText, error).
func parseFields(cmd *cobra.Command, defaultMethod string) ([]parsedField, string, error) {
	fieldFlags, _ := cmd.Flags().GetStringArray("field")

	if len(fieldFlags) > 0 {
		fields := make([]parsedField, 0, len(fieldFlags))
		for _, f := range fieldFlags {
			pf, err := parseFieldFlag(f, defaultMethod)
			if err != nil {
				return nil, "", err
			}
			fields = append(fields, pf)
		}
		return fields, "", nil
	}

	// Try reading from stdin
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// stdin is a terminal, no piped input
		return nil, "", nil
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, "", nil
	}

	var input fillYAMLInput
	if err := yaml.Unmarshal(data, &input); err != nil {
		return nil, "", fmt.Errorf("failed to parse YAML input: %w", err)
	}

	fields := make([]parsedField, 0, len(input.Fields))
	for _, f := range input.Fields {
		method := f.Method
		if method == "" {
			method = defaultMethod
		}
		fields = append(fields, parsedField{
			label:  f.Label,
			id:     f.ID,
			value:  f.Value,
			method: method,
		})
	}
	return fields, input.Submit, nil
}

// parseFieldFlag parses a single --field flag value.
// Supports "Label=value" and "id:42=value" formats.
func parseFieldFlag(s string, defaultMethod string) (parsedField, error) {
	eqIdx := strings.Index(s, "=")
	if eqIdx < 0 {
		return parsedField{}, fmt.Errorf("invalid --field %q: expected \"Label=value\" or \"id:42=value\"", s)
	}

	key := s[:eqIdx]
	value := s[eqIdx+1:]

	if strings.HasPrefix(key, "id:") {
		idStr := strings.TrimPrefix(key, "id:")
		var id int
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id <= 0 {
			return parsedField{}, fmt.Errorf("invalid --field %q: id must be a positive integer", s)
		}
		return parsedField{id: id, value: value, method: defaultMethod}, nil
	}

	if key == "" {
		return parsedField{}, fmt.Errorf("invalid --field %q: label cannot be empty", s)
	}

	return parsedField{label: key, value: value, method: defaultMethod}, nil
}

// executeFill implements the "fill" step type for the "do" batch command.
// It reads the tree once and fills multiple fields from the "fields" list.
func executeFill(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.Reader == nil {
		return StepResult{Action: "fill"}, fmt.Errorf("reader not available on this platform")
	}

	method := StringParam(params, "method", "set-value")
	submitText := StringParam(params, "submit", "")
	tabBetween := BoolParam(params, "tab-between", false)
	windowID := IntParam(params, "window-id", 0)
	pid := IntParam(params, "pid", 0)

	// Parse fields from the "fields" param (list of maps)
	fieldsRaw, ok := params["fields"]
	if !ok {
		return StepResult{Action: "fill"}, fmt.Errorf("fill step requires a \"fields\" list")
	}
	fieldsList, ok := fieldsRaw.([]interface{})
	if !ok {
		return StepResult{Action: "fill"}, fmt.Errorf("fill step \"fields\" must be a list")
	}

	var fields []parsedField
	for _, rawField := range fieldsList {
		fMap, ok := rawField.(map[string]interface{})
		if !ok {
			return StepResult{Action: "fill"}, fmt.Errorf("each field must be a map with label/id and value")
		}
		f := parsedField{
			label:  StringParam(fMap, "label", ""),
			id:     IntParam(fMap, "id", 0),
			value:  StringParam(fMap, "value", ""),
			method: StringParam(fMap, "method", method),
		}
		if f.label == "" && f.id == 0 {
			return StepResult{Action: "fill"}, fmt.Errorf("each field must have a \"label\" or \"id\"")
		}
		fields = append(fields, f)
	}

	if len(fields) == 0 {
		return StepResult{Action: "fill"}, fmt.Errorf("fill step requires at least one field")
	}

	// Read tree once
	elements, err := provider.Reader.ReadElements(platform.ReadOptions{
		App:      app,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
	})
	if err != nil {
		return StepResult{Action: "fill"}, fmt.Errorf("failed to read elements: %w", err)
	}

	// Fill fields
	filled := 0
	for i, f := range fields {
		result := fillOneField(provider, elements, f, app, window, windowID, pid, tabBetween && i > 0)
		if !result.OK {
			return StepResult{Action: "fill"}, fmt.Errorf("field %q: %s", result.Label, result.Error)
		}
		filled++
	}

	// Submit if requested
	if submitText != "" {
		elem, err := resolveElementByTextFromTree(elements, submitText, "", false, 0)
		if err != nil {
			return StepResult{Action: "fill"}, fmt.Errorf("submit element %q not found: %w", submitText, err)
		}
		if provider.Inputter != nil {
			cx := elem.Bounds[0] + elem.Bounds[2]/2
			cy := elem.Bounds[1] + elem.Bounds[3]/2
			if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
				return StepResult{Action: "fill"}, fmt.Errorf("failed to click submit: %w", err)
			}
		}
	}

	return StepResult{Action: "fill", Text: fmt.Sprintf("%d fields", filled)}, nil
}
