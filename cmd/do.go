package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// DoResult is the YAML output of a batch do command.
type DoResult struct {
	OK        bool         `yaml:"ok"                  json:"ok"`
	Action    string       `yaml:"action"              json:"action"`
	Steps     int          `yaml:"steps"               json:"steps"`
	Completed int          `yaml:"completed"           json:"completed"`
	Error     string       `yaml:"error,omitempty"     json:"error,omitempty"`
	Results   []StepResult `yaml:"results"             json:"results"`
	Display   []ElementInfo `yaml:"display,omitempty"  json:"display,omitempty"`
}

// StepResult is the output for a single step within a batch.
type StepResult struct {
	Step    int          `yaml:"step"               json:"step"`
	OK      bool         `yaml:"ok"                 json:"ok"`
	Action  string       `yaml:"action"             json:"action"`
	Error   string       `yaml:"error,omitempty"    json:"error,omitempty"`
	Target  *ElementInfo `yaml:"target,omitempty"   json:"target,omitempty"`
	Focused *ElementInfo `yaml:"focused,omitempty"  json:"focused,omitempty"`
	Text    string       `yaml:"text,omitempty"     json:"text,omitempty"`
	Key     string       `yaml:"key,omitempty"      json:"key,omitempty"`
	Elapsed string       `yaml:"elapsed,omitempty"  json:"elapsed,omitempty"`
	Match   string       `yaml:"match,omitempty"    json:"match,omitempty"`
	State   string       `yaml:"state,omitempty"    json:"state,omitempty"`
}

var doCmd = &cobra.Command{
	Use:   "do",
	Short: "Execute multiple actions in a batch",
	Long: `Execute a sequence of actions from a YAML list on stdin.

Each step is a command name with its flags as a map. Steps execute sequentially,
and by default execution stops on the first error.

Supported step types: click, type, action, set-value, scroll, wait, focus, read, sleep

Example:
  desktop-cli do --app "Safari" <<'EOF'
  - click: { text: "Full Name" }
  - type: { text: "John Doe" }
  - type: { key: "tab" }
  - type: { text: "john@example.com" }
  - click: { text: "Submit" }
  - wait: { for-text: "Thank you", timeout: 10 }
  EOF`,
	RunE: runDo,
}

func init() {
	rootCmd.AddCommand(doCmd)
	doCmd.Flags().String("app", "", "Default app for all steps (can be overridden per-step)")
	doCmd.Flags().String("window", "", "Default window for all steps")
	doCmd.Flags().Bool("stop-on-error", true, "Stop execution on first error (default: true)")
}

func runDo(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}

	defaultApp, _ := cmd.Flags().GetString("app")
	defaultWindow, _ := cmd.Flags().GetString("window")
	stopOnError, _ := cmd.Flags().GetBool("stop-on-error")

	// Read YAML steps from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("no steps provided on stdin — pipe a YAML list of actions")
	}

	var rawSteps []map[string]map[string]interface{}
	if err := yaml.Unmarshal(data, &rawSteps); err != nil {
		return fmt.Errorf("failed to parse YAML steps: %w", err)
	}

	if len(rawSteps) == 0 {
		return fmt.Errorf("no steps provided — expected a YAML list of actions")
	}

	results := make([]StepResult, 0, len(rawSteps))
	completed := 0
	hasFailure := false
	var lastErr string
	var lastApp string // track last app used for display elements

	for i, step := range rawSteps {
		stepNum := i + 1

		if len(step) != 1 {
			errMsg := fmt.Sprintf("step %d: expected exactly one action key, got %d", stepNum, len(step))
			hasFailure = true
			if stopOnError {
				results = append(results, StepResult{Step: stepNum, OK: false, Error: errMsg})
				lastErr = errMsg
				break
			}
			results = append(results, StepResult{Step: stepNum, OK: false, Error: errMsg})
			continue
		}

		for action, params := range step {
			app := stringParam(params, "app", defaultApp)
			window := stringParam(params, "window", defaultWindow)
			lastApp = app

			result, err := executeStep(provider, action, params, app, window)
			result.Step = stepNum
			if err != nil {
				result.OK = false
				result.Error = err.Error()
				results = append(results, result)
				hasFailure = true
				if stopOnError {
					lastErr = fmt.Sprintf("step %d: %s", stepNum, err.Error())
					goto done
				}
			} else {
				result.OK = true
				completed++
				results = append(results, result)
			}
		}
	}

done:
	if !hasFailure {
		completed = len(results)
	}

	// Collect display elements once at the end using the last app context
	var display []ElementInfo
	if lastApp != "" {
		display = readDisplayElements(provider, lastApp, defaultWindow, 0, 0)
	}

	allOK := !hasFailure
	return output.Print(DoResult{
		OK:        allOK,
		Action:    "do",
		Steps:     len(rawSteps),
		Completed: completed,
		Error:     lastErr,
		Results:   results,
		Display:   display,
	})
}

func executeStep(provider *platform.Provider, action string, params map[string]interface{}, app, window string) (StepResult, error) {
	switch action {
	case "click":
		return executeClick(provider, params, app, window)
	case "type":
		return executeType(provider, params, app, window)
	case "action":
		return executeAction(provider, params, app, window)
	case "set-value":
		return executeSetValue(provider, params, app, window)
	case "scroll":
		return executeScroll(provider, params, app, window)
	case "wait":
		return executeWait(provider, params, app, window)
	case "focus":
		return executeFocus(provider, params, app, window)
	case "read":
		return executeRead(provider, params, app, window)
	case "sleep":
		return executeSleep(params)
	default:
		return StepResult{Action: action}, fmt.Errorf("unknown step type %q — supported: click, type, action, set-value, scroll, wait, focus, read, sleep", action)
	}
}

func executeClick(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.Inputter == nil {
		return StepResult{Action: "click"}, fmt.Errorf("input not available on this platform")
	}

	text := stringParam(params, "text", "")
	id := intParam(params, "id", 0)
	x := intParam(params, "x", 0)
	y := intParam(params, "y", 0)
	buttonStr := stringParam(params, "button", "left")
	double := boolParam(params, "double", false)
	roles := stringParam(params, "roles", "")
	exact := boolParam(params, "exact", false)
	scopeID := intParam(params, "scope-id", 0)
	near := boolParam(params, "near", false)
	nearDirection := stringParam(params, "near-direction", "")

	button, err := platform.ParseMouseButton(buttonStr)
	if err != nil {
		return StepResult{Action: "click"}, err
	}

	count := 1
	if double {
		count = 2
	}

	var target *ElementInfo

	if text != "" {
		elem, allElements, err := resolveElementByText(provider, app, window, 0, 0, text, roles, exact, scopeID)
		if err != nil {
			return StepResult{Action: "click"}, err
		}
		if near {
			if nearest := findNearestInteractiveElement(allElements, elem, nearDirection); nearest != nil {
				elem = nearest
				target = elementInfoFromElement(elem)
				x = elem.Bounds[0] + elem.Bounds[2]/2
				y = elem.Bounds[1] + elem.Bounds[3]/2
			} else {
				x, y = nearFallbackOffset(elem, nearDirection)
			}
		} else {
			target = elementInfoFromElement(elem)
			x = elem.Bounds[0] + elem.Bounds[2]/2
			y = elem.Bounds[1] + elem.Bounds[3]/2
		}
	} else if id > 0 {
		if provider.Reader == nil {
			return StepResult{Action: "click"}, fmt.Errorf("reader not available on this platform")
		}
		elements, err := provider.Reader.ReadElements(platform.ReadOptions{App: app, Window: window})
		if err != nil {
			return StepResult{Action: "click"}, err
		}
		elem := findElementByID(elements, id)
		if elem == nil {
			return StepResult{Action: "click"}, fmt.Errorf("element with id %d not found", id)
		}
		target = elementInfoFromElement(elem)
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	} else if x == 0 && y == 0 {
		return StepResult{Action: "click"}, fmt.Errorf("specify text, id, or x/y coordinates")
	}

	if err := provider.Inputter.Click(x, y, button, count); err != nil {
		return StepResult{Action: "click"}, err
	}

	return StepResult{Action: "click", Target: target}, nil
}

func executeType(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.Inputter == nil {
		return StepResult{Action: "type"}, fmt.Errorf("input not available on this platform")
	}

	text := stringParam(params, "text", "")
	key := stringParam(params, "key", "")
	target := stringParam(params, "target", "")
	id := intParam(params, "id", 0)
	roles := stringParam(params, "roles", "")
	exact := boolParam(params, "exact", false)
	scopeID := intParam(params, "scope-id", 0)
	delayMs := intParam(params, "delay", 0)

	if text == "" && key == "" {
		return StepResult{Action: "type"}, fmt.Errorf("specify text or key")
	}

	hasTargetedElement := false

	// Focus element if --target or --id specified
	if target != "" {
		elem, _, err := resolveElementByText(provider, app, window, 0, 0, target, roles, exact, scopeID)
		if err != nil {
			return StepResult{Action: "type"}, err
		}
		hasTargetedElement = true
		cx := elem.Bounds[0] + elem.Bounds[2]/2
		cy := elem.Bounds[1] + elem.Bounds[3]/2
		if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
			return StepResult{Action: "type"}, fmt.Errorf("failed to focus element: %w", err)
		}
		time.Sleep(50 * time.Millisecond)
	} else if id > 0 {
		if provider.Reader == nil {
			return StepResult{Action: "type"}, fmt.Errorf("reader not available on this platform")
		}
		elements, err := provider.Reader.ReadElements(platform.ReadOptions{App: app, Window: window})
		if err != nil {
			return StepResult{Action: "type"}, fmt.Errorf("failed to read elements: %w", err)
		}
		elem := findElementByID(elements, id)
		if elem == nil {
			return StepResult{Action: "type"}, fmt.Errorf("element with id %d not found", id)
		}
		hasTargetedElement = true
		cx := elem.Bounds[0] + elem.Bounds[2]/2
		cy := elem.Bounds[1] + elem.Bounds[3]/2
		if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
			return StepResult{Action: "type"}, fmt.Errorf("failed to focus element: %w", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Type text
	if text != "" {
		if isCalculatorApp(app) && !hasTargetedElement && provider.ActionPerformer != nil {
			if err := typeViaButtons(provider, app, window, text); err != nil {
				return StepResult{Action: "type"}, err
			}
		} else {
			if err := provider.Inputter.TypeText(text, delayMs); err != nil {
				return StepResult{Action: "type"}, err
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Capture target info after typing but before key press
	var targetInfo *ElementInfo
	if hasTargetedElement {
		targetInfo = readFocusedElement(provider, app, window, 0, 0)
	}

	// Press key combo
	if key != "" {
		keys := strings.Split(key, "+")
		if err := provider.Inputter.KeyCombo(keys); err != nil {
			return StepResult{Action: "type"}, err
		}
		time.Sleep(80 * time.Millisecond)
	}

	result := StepResult{Text: text, Key: key}

	switch {
	case text != "" && key != "":
		result.Action = "type+key"
	case key != "":
		result.Action = "key"
	default:
		result.Action = "type"
	}

	if hasTargetedElement {
		result.Target = targetInfo
	} else if key != "" {
		result.Focused = readFocusedElement(provider, app, window, 0, 0)
	}

	return result, nil
}

func executeAction(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.ActionPerformer == nil {
		return StepResult{Action: "action"}, fmt.Errorf("action not supported on this platform")
	}

	id := intParam(params, "id", 0)
	actionName := stringParam(params, "action", "press")
	text := stringParam(params, "text", "")
	roles := stringParam(params, "roles", "")
	exact := boolParam(params, "exact", false)
	scopeID := intParam(params, "scope-id", 0)
	windowID := intParam(params, "window-id", 0)
	pid := intParam(params, "pid", 0)

	if id == 0 && text == "" {
		return StepResult{Action: "action"}, fmt.Errorf("specify id or text to target an element")
	}

	var preActionTarget *ElementInfo
	if text != "" && id == 0 {
		elem, _, err := resolveElementByText(provider, app, window, windowID, pid, text, roles, exact, scopeID)
		if err != nil {
			return StepResult{Action: "action"}, err
		}
		id = elem.ID
		preActionTarget = elementInfoFromElement(elem)
	} else {
		preActionTarget = readElementByID(provider, app, window, windowID, pid, id)
	}

	opts := platform.ActionOptions{
		App:      app,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
		ID:       id,
		Action:   actionName,
	}

	if err := provider.ActionPerformer.PerformAction(opts); err != nil {
		return StepResult{Action: "action"}, err
	}

	return StepResult{Action: "action", Target: preActionTarget}, nil
}

func executeSetValue(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.ValueSetter == nil {
		return StepResult{Action: "set-value"}, fmt.Errorf("set-value not supported on this platform")
	}

	id := intParam(params, "id", 0)
	value := stringParam(params, "value", "")
	attribute := stringParam(params, "attribute", "value")
	text := stringParam(params, "text", "")
	roles := stringParam(params, "roles", "")
	exact := boolParam(params, "exact", false)
	scopeID := intParam(params, "scope-id", 0)
	windowID := intParam(params, "window-id", 0)
	pid := intParam(params, "pid", 0)

	if id == 0 && text == "" {
		return StepResult{Action: "set-value"}, fmt.Errorf("specify id or text to target an element")
	}

	if text != "" && id == 0 {
		elem, _, err := resolveElementByText(provider, app, window, windowID, pid, text, roles, exact, scopeID)
		if err != nil {
			return StepResult{Action: "set-value"}, err
		}
		id = elem.ID
	}

	opts := platform.SetValueOptions{
		App:       app,
		Window:    window,
		WindowID:  windowID,
		PID:       pid,
		ID:        id,
		Value:     value,
		Attribute: attribute,
	}

	if err := provider.ValueSetter.SetValue(opts); err != nil {
		return StepResult{Action: "set-value"}, err
	}

	return StepResult{Action: "set-value"}, nil
}

func executeScroll(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.Inputter == nil {
		return StepResult{Action: "scroll"}, fmt.Errorf("input not available on this platform")
	}

	direction := stringParam(params, "direction", "")
	amount := intParam(params, "amount", 3)
	x := intParam(params, "x", 0)
	y := intParam(params, "y", 0)
	id := intParam(params, "id", 0)
	text := stringParam(params, "text", "")
	roles := stringParam(params, "roles", "")
	exact := boolParam(params, "exact", false)
	scopeID := intParam(params, "scope-id", 0)

	if direction == "" {
		return StepResult{Action: "scroll"}, fmt.Errorf("direction is required (up, down, left, right)")
	}

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
		return StepResult{Action: "scroll"}, fmt.Errorf("invalid direction %q: use up, down, left, or right", direction)
	}

	if text != "" {
		elem, _, err := resolveElementByText(provider, app, window, 0, 0, text, roles, exact, scopeID)
		if err != nil {
			return StepResult{Action: "scroll"}, err
		}
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	} else if id > 0 {
		if provider.Reader == nil {
			return StepResult{Action: "scroll"}, fmt.Errorf("reader not available on this platform")
		}
		elements, err := provider.Reader.ReadElements(platform.ReadOptions{App: app, Window: window})
		if err != nil {
			return StepResult{Action: "scroll"}, err
		}
		elem := findElementByID(elements, id)
		if elem == nil {
			return StepResult{Action: "scroll"}, fmt.Errorf("element with ID %d not found", id)
		}
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	}

	if err := provider.Inputter.Scroll(x, y, dx, dy); err != nil {
		return StepResult{Action: "scroll"}, err
	}

	return StepResult{Action: "scroll"}, nil
}

func executeWait(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.Reader == nil {
		return StepResult{Action: "wait"}, fmt.Errorf("reader not available on this platform")
	}

	forText := stringParam(params, "for-text", "")
	forRole := stringParam(params, "for-role", "")
	forID := intParam(params, "for-id", 0)
	gone := boolParam(params, "gone", false)
	timeoutSec := intParam(params, "timeout", 30)
	intervalMs := intParam(params, "interval", 500)
	pid := intParam(params, "pid", 0)
	windowID := intParam(params, "window-id", 0)

	if forText == "" && forRole == "" && forID == 0 {
		return StepResult{Action: "wait"}, fmt.Errorf("specify at least one condition: for-text, for-role, or for-id")
	}

	var forRoles []string
	if forRole != "" {
		forRoles = []string{forRole}
	}

	readOpts := platform.ReadOptions{
		App:      app,
		Window:   window,
		PID:      pid,
		WindowID: windowID,
	}

	timeout := time.Duration(timeoutSec) * time.Second
	interval := time.Duration(intervalMs) * time.Millisecond
	deadline := time.Now().Add(timeout)
	start := time.Now()

	for {
		elements, err := provider.Reader.ReadElements(readOpts)
		if err != nil {
			if time.Now().After(deadline) {
				return StepResult{Action: "wait"}, fmt.Errorf("timeout after %s (last error: %w)", timeout, err)
			}
			time.Sleep(interval)
			continue
		}

		matched := checkWaitCondition(elements, forText, forRoles, forID)
		conditionMet := matched
		if gone {
			conditionMet = !matched
		}

		if conditionMet {
			elapsed := time.Since(start)
			matchDesc := describeCondition(forText, forRole, forID, gone)
			return StepResult{
				Action:  "wait",
				Elapsed: fmt.Sprintf("%.1fs", elapsed.Seconds()),
				Match:   matchDesc,
			}, nil
		}

		if time.Now().After(deadline) {
			matchDesc := describeCondition(forText, forRole, forID, gone)
			return StepResult{Action: "wait"}, fmt.Errorf("timed out waiting for condition: %s", matchDesc)
		}

		time.Sleep(interval)
	}
}

func executeFocus(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.WindowManager == nil {
		return StepResult{Action: "focus"}, fmt.Errorf("window management not available on this platform")
	}

	windowID := intParam(params, "window-id", 0)
	pid := intParam(params, "pid", 0)

	if app == "" && window == "" && windowID == 0 && pid == 0 {
		return StepResult{Action: "focus"}, fmt.Errorf("specify app, window, window-id, or pid")
	}

	opts := platform.FocusOptions{
		App:      app,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
	}

	if err := provider.WindowManager.FocusWindow(opts); err != nil {
		return StepResult{Action: "focus"}, err
	}

	return StepResult{Action: "focus"}, nil
}

func executeRead(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.Reader == nil {
		return StepResult{Action: "read"}, fmt.Errorf("reader not available on this platform")
	}

	formatStr := stringParam(params, "format", "agent")
	pid := intParam(params, "pid", 0)
	windowID := intParam(params, "window-id", 0)

	readOpts := platform.ReadOptions{
		App:      app,
		Window:   window,
		PID:      pid,
		WindowID: windowID,
		Depth:    intParam(params, "depth", 0),
	}

	elements, err := provider.Reader.ReadElements(readOpts)
	if err != nil {
		return StepResult{Action: "read"}, err
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

	var state string
	if formatStr == "agent" {
		state = output.FormatAgentString(app, pid, windowTitle, elements)
	}

	return StepResult{Action: "read", State: state}, nil
}

func executeSleep(params map[string]interface{}) (StepResult, error) {
	ms := intParam(params, "ms", 0)
	if ms <= 0 {
		return StepResult{Action: "sleep"}, fmt.Errorf("ms must be > 0")
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return StepResult{Action: "sleep", Elapsed: fmt.Sprintf("%dms", ms)}, nil
}

// Parameter extraction helpers for step maps

func stringParam(params map[string]interface{}, key, defaultVal string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		// Handle numeric values that YAML may parse as int/float
		return fmt.Sprintf("%v", v)
	}
	return defaultVal
}

func intParam(params map[string]interface{}, key string, defaultVal int) int {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		case int64:
			return int(n)
		}
	}
	return defaultVal
}

func boolParam(params map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}
