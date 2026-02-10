package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mj1618/desktop-cli/internal/model"
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
	Step        int          `yaml:"step"                   json:"step"`
	OK          bool         `yaml:"ok"                     json:"ok"`
	Action      string       `yaml:"action"                 json:"action"`
	Error       string       `yaml:"error,omitempty"        json:"error,omitempty"`
	Target      *ElementInfo `yaml:"target,omitempty"       json:"target,omitempty"`
	Focused     *ElementInfo `yaml:"focused,omitempty"      json:"focused,omitempty"`
	Text        string       `yaml:"text,omitempty"         json:"text,omitempty"`
	Key         string       `yaml:"key,omitempty"          json:"key,omitempty"`
	Elapsed     string       `yaml:"elapsed,omitempty"      json:"elapsed,omitempty"`
	Match       string       `yaml:"match,omitempty"        json:"match,omitempty"`
	State       string       `yaml:"state,omitempty"        json:"state,omitempty"`
	Matched     *bool        `yaml:"matched,omitempty"      json:"matched,omitempty"`
	Branch      string       `yaml:"branch,omitempty"       json:"branch,omitempty"`
	Substeps    []StepResult `yaml:"substeps,omitempty"     json:"substeps,omitempty"`
	Verified    *bool        `yaml:"verified,omitempty"     json:"verified,omitempty"`
	Retried     *bool        `yaml:"retried,omitempty"      json:"retried,omitempty"`
	RetryMethod string       `yaml:"retry_method,omitempty" json:"retry_method,omitempty"`
	RetryReason string       `yaml:"retry_reason,omitempty" json:"retry_reason,omitempty"`
}

var doCmd = &cobra.Command{
	Use:   "do",
	Short: "Execute multiple actions in a batch",
	Long: `Execute a sequence of actions from a YAML list on stdin.

Each step is a command name with its flags as a map. Steps execute sequentially,
and by default execution stops on the first error.

Supported step types: click, hover, type, action, set-value, fill, scroll, wait, assert, focus, read, open, sleep, if-exists, if-focused, try

Conditional step types:
  if-exists: { text: "Accept" }     # check if element exists
    then: [steps]                    # run if found
    else: [steps]                    # run if not found (optional)
  if-focused: { roles: "input" }    # check if focused element matches
    then: [steps]
    else: [steps]
  try: [steps]                       # run steps, continue even if they fail

Example:
  desktop-cli do --app "Safari" <<'EOF'
  - try:
    - click: { text: "Accept Cookies" }
  - if-exists: { text: "Sign In", roles: "btn" }
    then:
    - click: { text: "Sign In" }
    else:
    - read: { format: "agent" }
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

	var rawSteps []map[string]interface{}
	if err := yaml.Unmarshal(data, &rawSteps); err != nil {
		return fmt.Errorf("failed to parse YAML steps: %w", err)
	}

	if len(rawSteps) == 0 {
		return fmt.Errorf("no steps provided — expected a YAML list of actions")
	}

	ctx := &DoContext{
		Provider:      provider,
		DefaultApp:    defaultApp,
		DefaultWindow: defaultWindow,
		StopOnError:   stopOnError,
	}

	ctx.ExecuteSteps(rawSteps, 0)

	results := ctx.Results
	hasFailure := ctx.HasFailure
	completed := 0
	var lastErr string
	if hasFailure {
		for _, r := range results {
			if r.OK {
				completed++
			}
		}
		if len(results) > 0 && !results[len(results)-1].OK && results[len(results)-1].Error != "" {
			lastErr = fmt.Sprintf("step %d: %s", results[len(results)-1].Step, results[len(results)-1].Error)
		}
	} else {
		completed = len(results)
	}
	lastApp := ctx.LastApp

	// Collect display elements once at the end using the last app context
	var display []ElementInfo
	if lastApp != "" {
		display = readDisplayElements(provider, lastApp, defaultWindow, 0, 0, [4]int{})
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

// DoContext tracks state during batch step execution.
// Exported for use by the MCP server.
type DoContext struct {
	Provider      *platform.Provider
	DefaultApp    string
	DefaultWindow string
	StopOnError   bool
	Results       []StepResult
	HasFailure    bool
	Stopped       bool
	LastApp       string
}

// executeSteps runs a list of raw YAML steps, appending results to ctx.Results.
// stepOffset is added to step numbers for substep numbering.
func (ctx *DoContext) ExecuteSteps(rawSteps []map[string]interface{}, stepOffset int) {
	for i, step := range rawSteps {
		if ctx.Stopped {
			break
		}
		stepNum := stepOffset + i + 1

		// Check for conditional step types
		if _, ok := step["if-exists"]; ok {
			ctx.executeIfExists(step, stepNum)
			continue
		}
		if _, ok := step["if-focused"]; ok {
			ctx.executeIfFocused(step, stepNum)
			continue
		}
		if _, ok := step["try"]; ok {
			ctx.executeTry(step, stepNum)
			continue
		}

		// Regular step: find the single action key
		action, params, err := parseRegularStep(step)
		if err != nil {
			ctx.HasFailure = true
			if ctx.StopOnError {
				ctx.Results = append(ctx.Results, StepResult{Step: stepNum, OK: false, Error: err.Error()})
				ctx.Stopped = true
				break
			}
			ctx.Results = append(ctx.Results, StepResult{Step: stepNum, OK: false, Error: err.Error()})
			continue
		}

		app := StringParam(params, "app", ctx.DefaultApp)
		window := StringParam(params, "window", ctx.DefaultWindow)
		ctx.LastApp = app

		result, execErr := ExecuteStep(ctx.Provider, action, params, app, window)
		result.Step = stepNum
		if execErr != nil {
			result.OK = false
			result.Error = execErr.Error()
			ctx.Results = append(ctx.Results, result)
			ctx.HasFailure = true
			if ctx.StopOnError {
				ctx.Stopped = true
				break
			}
		} else {
			result.OK = true
			ctx.Results = append(ctx.Results, result)
		}
	}
}

// parseRegularStep extracts the action name and params from a regular (non-conditional) step.
func parseRegularStep(step map[string]interface{}) (string, map[string]interface{}, error) {
	// Find the action key (skip "then", "else" which belong to conditionals)
	var action string
	var paramsRaw interface{}
	for k, v := range step {
		if k == "then" || k == "else" {
			continue
		}
		if action != "" {
			return "", nil, fmt.Errorf("expected exactly one action key, got multiple")
		}
		action = k
		paramsRaw = v
	}
	if action == "" {
		return "", nil, fmt.Errorf("no action key found in step")
	}

	params, ok := paramsRaw.(map[string]interface{})
	if !ok {
		// Handle nil params (e.g. `- sleep:` with no value)
		params = make(map[string]interface{})
	}
	return action, params, nil
}

// parseSubsteps converts a YAML value (expected to be []interface{}) into []map[string]interface{}.
func parseSubsteps(raw interface{}) ([]map[string]interface{}, error) {
	if raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected array of steps, got %T", raw)
	}
	steps := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected step map, got %T", item)
		}
		steps = append(steps, m)
	}
	return steps, nil
}

// executeIfExists handles the if-exists conditional step.
func (ctx *DoContext) executeIfExists(step map[string]interface{}, stepNum int) {
	condRaw := step["if-exists"]
	condParams, ok := condRaw.(map[string]interface{})
	if !ok {
		ctx.HasFailure = true
		r := StepResult{Step: stepNum, OK: false, Action: "if-exists", Error: "if-exists condition must be a map with text/roles/id params"}
		ctx.Results = append(ctx.Results, r)
		if ctx.StopOnError {
			ctx.Stopped = true
		}
		return
	}

	// Check if the element exists
	text := StringParam(condParams, "text", "")
	id := IntParam(condParams, "id", 0)
	roles := StringParam(condParams, "roles", "")
	exact := BoolParam(condParams, "exact", false)
	scopeID := IntParam(condParams, "scope-id", 0)
	app := StringParam(condParams, "app", ctx.DefaultApp)
	window := StringParam(condParams, "window", ctx.DefaultWindow)
	ctx.LastApp = app

	matched := false
	if text != "" && ctx.Provider != nil {
		elem, _, err := resolveElementByText(ctx.Provider, app, window, 0, 0, text, roles, exact, scopeID)
		matched = err == nil && elem != nil
	} else if id > 0 {
		if ctx.Provider.Reader != nil {
			elements, err := ctx.Provider.Reader.ReadElements(platform.ReadOptions{App: app, Window: window})
			if err == nil {
				matched = findElementByID(elements, id) != nil
			}
		}
	}

	matchedVal := matched
	result := StepResult{
		Step:    stepNum,
		OK:      true,
		Action:  "if-exists",
		Matched: &matchedVal,
	}

	// Determine which branch to execute
	var substeps []map[string]interface{}
	if matched {
		result.Branch = "then"
		if thenRaw, ok := step["then"]; ok {
			var err error
			substeps, err = parseSubsteps(thenRaw)
			if err != nil {
				result.OK = false
				result.Error = fmt.Sprintf("invalid then steps: %s", err)
				ctx.HasFailure = true
				ctx.Results = append(ctx.Results, result)
				if ctx.StopOnError {
					ctx.Stopped = true
				}
				return
			}
		}
	} else {
		result.Branch = "else"
		if elseRaw, ok := step["else"]; ok {
			var err error
			substeps, err = parseSubsteps(elseRaw)
			if err != nil {
				result.OK = false
				result.Error = fmt.Sprintf("invalid else steps: %s", err)
				ctx.HasFailure = true
				ctx.Results = append(ctx.Results, result)
				if ctx.StopOnError {
					ctx.Stopped = true
				}
				return
			}
		}
	}

	// Execute the selected branch, collecting substep results
	if len(substeps) > 0 {
		subCtx := &DoContext{
			Provider:      ctx.Provider,
			DefaultApp:    ctx.DefaultApp,
			DefaultWindow: ctx.DefaultWindow,
			StopOnError:   ctx.StopOnError,
			LastApp:       ctx.LastApp,
		}
		subCtx.ExecuteSteps(substeps, 0)
		result.Substeps = subCtx.Results
		ctx.LastApp = subCtx.LastApp
		if subCtx.HasFailure {
			ctx.HasFailure = true
			result.OK = false
			if ctx.StopOnError {
				ctx.Stopped = true
			}
		}
	}

	ctx.Results = append(ctx.Results, result)
}

// executeIfFocused handles the if-focused conditional step.
func (ctx *DoContext) executeIfFocused(step map[string]interface{}, stepNum int) {
	condRaw := step["if-focused"]
	condParams, ok := condRaw.(map[string]interface{})
	if !ok {
		ctx.HasFailure = true
		r := StepResult{Step: stepNum, OK: false, Action: "if-focused", Error: "if-focused condition must be a map with roles/text params"}
		ctx.Results = append(ctx.Results, r)
		if ctx.StopOnError {
			ctx.Stopped = true
		}
		return
	}

	roles := StringParam(condParams, "roles", "")
	text := StringParam(condParams, "text", "")
	app := StringParam(condParams, "app", ctx.DefaultApp)
	window := StringParam(condParams, "window", ctx.DefaultWindow)
	ctx.LastApp = app

	// Read the focused element
	var focusedInfo *ElementInfo
	if ctx.Provider != nil {
		focusedInfo = readFocusedElement(ctx.Provider, app, window, 0, 0)
	}

	matched := false
	if focusedInfo != nil {
		matched = true
		// Check role filter
		if roles != "" {
			roleSet := make(map[string]bool)
			for _, r := range strings.Split(roles, ",") {
				r = strings.TrimSpace(r)
				if r != "" {
					roleSet[r] = true
				}
			}
			if !roleSet[focusedInfo.Role] {
				matched = false
			}
		}
		// Check text filter
		if text != "" && matched {
			textLower := strings.ToLower(text)
			if !strings.Contains(strings.ToLower(focusedInfo.Title), textLower) &&
				!strings.Contains(strings.ToLower(focusedInfo.Value), textLower) &&
				!strings.Contains(strings.ToLower(focusedInfo.Description), textLower) {
				matched = false
			}
		}
	}

	matchedVal := matched
	result := StepResult{
		Step:    stepNum,
		OK:      true,
		Action:  "if-focused",
		Matched: &matchedVal,
		Focused: focusedInfo,
	}

	var substeps []map[string]interface{}
	if matched {
		result.Branch = "then"
		if thenRaw, ok := step["then"]; ok {
			var err error
			substeps, err = parseSubsteps(thenRaw)
			if err != nil {
				result.OK = false
				result.Error = fmt.Sprintf("invalid then steps: %s", err)
				ctx.HasFailure = true
				ctx.Results = append(ctx.Results, result)
				if ctx.StopOnError {
					ctx.Stopped = true
				}
				return
			}
		}
	} else {
		result.Branch = "else"
		if elseRaw, ok := step["else"]; ok {
			var err error
			substeps, err = parseSubsteps(elseRaw)
			if err != nil {
				result.OK = false
				result.Error = fmt.Sprintf("invalid else steps: %s", err)
				ctx.HasFailure = true
				ctx.Results = append(ctx.Results, result)
				if ctx.StopOnError {
					ctx.Stopped = true
				}
				return
			}
		}
	}

	if len(substeps) > 0 {
		subCtx := &DoContext{
			Provider:      ctx.Provider,
			DefaultApp:    ctx.DefaultApp,
			DefaultWindow: ctx.DefaultWindow,
			StopOnError:   ctx.StopOnError,
			LastApp:       ctx.LastApp,
		}
		subCtx.ExecuteSteps(substeps, 0)
		result.Substeps = subCtx.Results
		ctx.LastApp = subCtx.LastApp
		if subCtx.HasFailure {
			ctx.HasFailure = true
			result.OK = false
			if ctx.StopOnError {
				ctx.Stopped = true
			}
		}
	}

	ctx.Results = append(ctx.Results, result)
}

// executeTry handles the try step type — executes substeps and always continues.
func (ctx *DoContext) executeTry(step map[string]interface{}, stepNum int) {
	tryRaw := step["try"]
	substeps, err := parseSubsteps(tryRaw)
	if err != nil {
		ctx.HasFailure = true
		r := StepResult{Step: stepNum, OK: false, Action: "try", Error: fmt.Sprintf("invalid try steps: %s", err)}
		ctx.Results = append(ctx.Results, r)
		if ctx.StopOnError {
			ctx.Stopped = true
		}
		return
	}

	// Execute substeps with stopOnError=true (stop within the try block on first error)
	// but the try block itself always succeeds
	subCtx := &DoContext{
		Provider:      ctx.Provider,
		DefaultApp:    ctx.DefaultApp,
		DefaultWindow: ctx.DefaultWindow,
		StopOnError:   true, // stop within try on first error
		LastApp:       ctx.LastApp,
	}
	subCtx.ExecuteSteps(substeps, 0)

	result := StepResult{
		Step:     stepNum,
		OK:       true, // try blocks always succeed
		Action:   "try",
		Substeps: subCtx.Results,
	}
	ctx.LastApp = subCtx.LastApp

	ctx.Results = append(ctx.Results, result)
}

// ExecuteStep dispatches a single action step to the appropriate handler.
// Used by both the `do` batch command and the MCP server.
func ExecuteStep(provider *platform.Provider, action string, params map[string]interface{}, app, window string) (StepResult, error) {
	switch action {
	case "click":
		return ExecuteClick(provider, params, app, window)
	case "hover":
		return ExecuteHover(provider, params, app, window)
	case "type":
		return ExecuteType(provider, params, app, window)
	case "action":
		return ExecuteAction(provider, params, app, window)
	case "set-value":
		return ExecuteSetValue(provider, params, app, window)
	case "scroll":
		return ExecuteScroll(provider, params, app, window)
	case "wait":
		return ExecuteWait(provider, params, app, window)
	case "focus":
		return ExecuteFocus(provider, params, app, window)
	case "read":
		return ExecuteRead(provider, params, app, window)
	case "open":
		return ExecuteOpen(params)
	case "assert":
		return ExecuteAssert(provider, params, app, window)
	case "fill":
		return executeFill(provider, params, app, window)
	case "sleep":
		return ExecuteSleep(params)
	default:
		return StepResult{Action: action}, fmt.Errorf("unknown step type %q — supported: click, hover, type, action, set-value, fill, scroll, wait, focus, read, open, assert, sleep, if-exists, if-focused, try", action)
	}
}

// getVerifyOptionsFromParams extracts verify options from a do-step params map.
func getVerifyOptionsFromParams(params map[string]interface{}) verifyOptions {
	return verifyOptions{
		Verify:      BoolParam(params, "verify", false),
		VerifyDelay: IntParam(params, "verify-delay", 100),
		MaxRetries:  IntParam(params, "max-retries", 2),
	}
}

// applyVerifyResult copies verification fields from a verifyResult to a StepResult.
func applyVerifyResult(sr *StepResult, vr verifyResult) {
	sr.Verified = boolPtr(vr.Verified)
	if vr.Retried {
		sr.Retried = boolPtr(true)
		sr.RetryMethod = vr.RetryMethod
		sr.RetryReason = vr.RetryReason
	}
}

func ExecuteClick(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.Inputter == nil {
		return StepResult{Action: "click"}, fmt.Errorf("input not available on this platform")
	}

	text := StringParam(params, "text", "")
	id := IntParam(params, "id", 0)
	x := IntParam(params, "x", 0)
	y := IntParam(params, "y", 0)
	buttonStr := StringParam(params, "button", "left")
	double := BoolParam(params, "double", false)
	roles := StringParam(params, "roles", "")
	exact := BoolParam(params, "exact", false)
	scopeID := IntParam(params, "scope-id", 0)
	near := BoolParam(params, "near", false)
	nearDirection := StringParam(params, "near-direction", "")

	button, err := platform.ParseMouseButton(buttonStr)
	if err != nil {
		return StepResult{Action: "click"}, err
	}

	count := 1
	if double {
		count = 2
	}

	vOpts := getVerifyOptionsFromParams(params)
	ref := StringParam(params, "ref", "")
	var target *ElementInfo
	var resolvedElem *model.Element
	var preSnapshot elementSnapshot

	if ref != "" {
		elem, _, err := resolveElementByRef(provider, app, window, 0, 0, ref)
		if err != nil {
			return StepResult{Action: "click"}, err
		}
		target = elementInfoFromElement(elem)
		resolvedElem = elem
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	} else if text != "" {
		if near {
			// --near mode: get ALL text matches, pick the best one for proximity search.
			allMatches, allElements, err := resolveAllTextMatches(provider, app, window, 0, 0, text, roles, exact, scopeID)
			if err != nil {
				return StepResult{Action: "click"}, err
			}
			elem := pickBestNearMatch(allElements, allMatches)
			resolvedElem = elem
			if nearest := findNearestInteractiveElement(allElements, elem, nearDirection); nearest != nil {
				elem = nearest
				resolvedElem = elem
				target = elementInfoFromElement(elem)
				x = elem.Bounds[0] + elem.Bounds[2]/2
				y = elem.Bounds[1] + elem.Bounds[3]/2
			} else {
				x, y = nearFallbackOffset(elem, nearDirection)
			}
		} else {
			elem, _, err := resolveElementByText(provider, app, window, 0, 0, text, roles, exact, scopeID)
			if err != nil {
				return StepResult{Action: "click"}, err
			}
			resolvedElem = elem
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
		resolvedElem = elem
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	} else if x == 0 && y == 0 {
		return StepResult{Action: "click"}, fmt.Errorf("specify text, id, or x/y coordinates")
	}

	if vOpts.Verify && resolvedElem != nil {
		preSnapshot = snapshotElement(resolvedElem)
	}

	if err := provider.Inputter.Click(x, y, button, count); err != nil {
		return StepResult{Action: "click"}, err
	}

	result := StepResult{Action: "click", Target: target}

	if vOpts.Verify && preSnapshot.Exists {
		var fallbacks []fallbackAction
		if provider.ActionPerformer != nil && resolvedElem != nil {
			elemID := resolvedElem.ID
			fallbacks = append(fallbacks, fallbackAction{
				Method: "action",
				Execute: func() error {
					return provider.ActionPerformer.PerformAction(platform.ActionOptions{
						App: app, Window: window, ID: elemID, Action: "press",
					})
				},
			})
		}
		fallbacks = append(fallbacks, fallbackAction{
			Method: "offset-click",
			Execute: func() error {
				return provider.Inputter.Click(x+2, y+2, button, count)
			},
		})
		vr := verifyAction(provider, preSnapshot, vOpts, app, window, 0, 0, fallbacks, 0)
		applyVerifyResult(&result, vr)
	}

	return result, nil
}

func ExecuteHover(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.Inputter == nil {
		return StepResult{Action: "hover"}, fmt.Errorf("input not available on this platform")
	}

	text := StringParam(params, "text", "")
	id := IntParam(params, "id", 0)
	x := IntParam(params, "x", 0)
	y := IntParam(params, "y", 0)
	roles := StringParam(params, "roles", "")
	exact := BoolParam(params, "exact", false)
	scopeID := IntParam(params, "scope-id", 0)

	ref := StringParam(params, "ref", "")
	var target *ElementInfo

	if ref != "" {
		elem, _, err := resolveElementByRef(provider, app, window, 0, 0, ref)
		if err != nil {
			return StepResult{Action: "hover"}, err
		}
		target = elementInfoFromElement(elem)
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	} else if text != "" {
		elem, _, err := resolveElementByText(provider, app, window, 0, 0, text, roles, exact, scopeID)
		if err != nil {
			return StepResult{Action: "hover"}, err
		}
		target = elementInfoFromElement(elem)
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	} else if id > 0 {
		if provider.Reader == nil {
			return StepResult{Action: "hover"}, fmt.Errorf("reader not available on this platform")
		}
		elements, err := provider.Reader.ReadElements(platform.ReadOptions{App: app, Window: window})
		if err != nil {
			return StepResult{Action: "hover"}, err
		}
		elem := findElementByID(elements, id)
		if elem == nil {
			return StepResult{Action: "hover"}, fmt.Errorf("element with id %d not found", id)
		}
		target = elementInfoFromElement(elem)
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	} else if x == 0 && y == 0 {
		return StepResult{Action: "hover"}, fmt.Errorf("specify text, ref, id, or x/y coordinates")
	}

	if err := provider.Inputter.MoveMouse(x, y); err != nil {
		return StepResult{Action: "hover"}, err
	}

	return StepResult{Action: "hover", Target: target}, nil
}

func ExecuteType(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.Inputter == nil {
		return StepResult{Action: "type"}, fmt.Errorf("input not available on this platform")
	}

	text := StringParam(params, "text", "")
	key := StringParam(params, "key", "")
	target := StringParam(params, "target", "")
	refParam := StringParam(params, "ref", "")
	id := IntParam(params, "id", 0)
	roles := StringParam(params, "roles", "")
	exact := BoolParam(params, "exact", false)
	scopeID := IntParam(params, "scope-id", 0)
	delayMs := IntParam(params, "delay", 0)

	if text == "" && key == "" {
		return StepResult{Action: "type"}, fmt.Errorf("specify text or key")
	}

	vOpts := getVerifyOptionsFromParams(params)
	hasTargetedElement := false
	var verifyElemID int

	// Focus element if ref, target, or id specified
	if refParam != "" {
		elem, _, err := resolveElementByRef(provider, app, window, 0, 0, refParam)
		if err != nil {
			return StepResult{Action: "type"}, err
		}
		hasTargetedElement = true
		verifyElemID = elem.ID
		cx := elem.Bounds[0] + elem.Bounds[2]/2
		cy := elem.Bounds[1] + elem.Bounds[3]/2
		if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
			return StepResult{Action: "type"}, fmt.Errorf("failed to focus element: %w", err)
		}
		time.Sleep(50 * time.Millisecond)
	} else if target != "" {
		elem, _, err := resolveElementByText(provider, app, window, 0, 0, target, roles, exact, scopeID)
		if err != nil {
			return StepResult{Action: "type"}, err
		}
		hasTargetedElement = true
		verifyElemID = elem.ID
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
		verifyElemID = elem.ID
		cx := elem.Bounds[0] + elem.Bounds[2]/2
		cy := elem.Bounds[1] + elem.Bounds[3]/2
		if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
			return StepResult{Action: "type"}, fmt.Errorf("failed to focus element: %w", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Snapshot focused element for verification (after click-to-focus, before typing)
	var preSnapshot elementSnapshot
	if vOpts.Verify && hasTargetedElement && provider.Reader != nil {
		elements, readErr := provider.Reader.ReadElements(platform.ReadOptions{App: app, Window: window})
		if readErr == nil {
			if focused := findFocusedElementRaw(elements); focused != nil {
				preSnapshot = snapshotElement(focused)
				verifyElemID = focused.ID
			} else if el := findElementByID(elements, verifyElemID); el != nil {
				preSnapshot = snapshotElement(el)
			}
		}
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

	if vOpts.Verify && preSnapshot.Exists {
		var fallbacks []fallbackAction
		if text != "" && provider.ValueSetter != nil {
			elemID := verifyElemID
			fallbacks = append(fallbacks, fallbackAction{
				Method: "set-value",
				Execute: func() error {
					return provider.ValueSetter.SetValue(platform.SetValueOptions{
						App: app, Window: window,
						ID: elemID, Value: text, Attribute: "value",
					})
				},
			})
		}
		vr := verifyAction(provider, preSnapshot, vOpts, app, window, 0, 0, fallbacks, 0)
		applyVerifyResult(&result, vr)
	}

	return result, nil
}

func ExecuteAction(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.ActionPerformer == nil {
		return StepResult{Action: "action"}, fmt.Errorf("action not supported on this platform")
	}

	id := IntParam(params, "id", 0)
	actionName := StringParam(params, "action", "press")
	text := StringParam(params, "text", "")
	ref := StringParam(params, "ref", "")
	roles := StringParam(params, "roles", "")
	exact := BoolParam(params, "exact", false)
	scopeID := IntParam(params, "scope-id", 0)
	windowID := IntParam(params, "window-id", 0)
	pid := IntParam(params, "pid", 0)

	if id == 0 && text == "" && ref == "" {
		return StepResult{Action: "action"}, fmt.Errorf("specify id, text, or ref to target an element")
	}

	vOpts := getVerifyOptionsFromParams(params)
	var preSnapshot elementSnapshot
	var preActionTarget *ElementInfo

	if ref != "" && id == 0 {
		elem, _, err := resolveElementByRef(provider, app, window, windowID, pid, ref)
		if err != nil {
			return StepResult{Action: "action"}, err
		}
		id = elem.ID
		preActionTarget = elementInfoFromElement(elem)
		if vOpts.Verify {
			preSnapshot = snapshotElement(elem)
		}
	} else if text != "" && id == 0 {
		elem, _, err := resolveElementByText(provider, app, window, windowID, pid, text, roles, exact, scopeID)
		if err != nil {
			return StepResult{Action: "action"}, err
		}
		id = elem.ID
		preActionTarget = elementInfoFromElement(elem)
		if vOpts.Verify {
			preSnapshot = snapshotElement(elem)
		}
	} else {
		preActionTarget = readElementByID(provider, app, window, windowID, pid, id)
		if vOpts.Verify && provider.Reader != nil {
			if elements, readErr := provider.Reader.ReadElements(platform.ReadOptions{
				App: app, Window: window, WindowID: windowID, PID: pid,
			}); readErr == nil {
				if el := findElementByID(elements, id); el != nil {
					preSnapshot = snapshotElement(el)
				}
			}
		}
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

	result := StepResult{Action: "action", Target: preActionTarget}

	if vOpts.Verify && preSnapshot.Exists {
		vr := verifyAction(provider, preSnapshot, vOpts, app, window, windowID, pid, nil, 0)
		applyVerifyResult(&result, vr)
	}

	return result, nil
}

func ExecuteSetValue(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.ValueSetter == nil {
		return StepResult{Action: "set-value"}, fmt.Errorf("set-value not supported on this platform")
	}

	id := IntParam(params, "id", 0)
	value := StringParam(params, "value", "")
	attribute := StringParam(params, "attribute", "value")
	text := StringParam(params, "text", "")
	ref := StringParam(params, "ref", "")
	roles := StringParam(params, "roles", "")
	exact := BoolParam(params, "exact", false)
	scopeID := IntParam(params, "scope-id", 0)
	windowID := IntParam(params, "window-id", 0)
	pid := IntParam(params, "pid", 0)

	if id == 0 && text == "" && ref == "" {
		return StepResult{Action: "set-value"}, fmt.Errorf("specify id, text, or ref to target an element")
	}

	vOpts := getVerifyOptionsFromParams(params)
	var resolvedElem *model.Element
	var preSnapshot elementSnapshot

	if ref != "" && id == 0 {
		elem, _, err := resolveElementByRef(provider, app, window, windowID, pid, ref)
		if err != nil {
			return StepResult{Action: "set-value"}, err
		}
		id = elem.ID
		resolvedElem = elem
	} else if text != "" && id == 0 {
		elem, _, err := resolveElementByText(provider, app, window, windowID, pid, text, roles, exact, scopeID)
		if err != nil {
			return StepResult{Action: "set-value"}, err
		}
		id = elem.ID
		resolvedElem = elem
	} else if id > 0 && provider.Reader != nil {
		if elements, readErr := provider.Reader.ReadElements(platform.ReadOptions{
			App: app, Window: window, WindowID: windowID, PID: pid,
		}); readErr == nil {
			resolvedElem = findElementByID(elements, id)
		}
	}

	if vOpts.Verify && resolvedElem != nil {
		preSnapshot = snapshotElement(resolvedElem)
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

	result := StepResult{Action: "set-value"}

	if vOpts.Verify && preSnapshot.Exists {
		var fallbacks []fallbackAction
		if provider.Inputter != nil && resolvedElem != nil && attribute == "value" {
			elemBounds := resolvedElem.Bounds
			fallbacks = append(fallbacks, fallbackAction{
				Method: "type",
				Execute: func() error {
					cx := elemBounds[0] + elemBounds[2]/2
					cy := elemBounds[1] + elemBounds[3]/2
					if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
						return err
					}
					time.Sleep(50 * time.Millisecond)
					if err := provider.Inputter.KeyCombo([]string{"cmd", "a"}); err != nil {
						return err
					}
					time.Sleep(30 * time.Millisecond)
					return provider.Inputter.TypeText(value, 0)
				},
			})
		}
		vr := verifyAction(provider, preSnapshot, vOpts, app, window, windowID, pid, fallbacks, 0)
		applyVerifyResult(&result, vr)
	}

	return result, nil
}

func ExecuteScroll(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.Inputter == nil {
		return StepResult{Action: "scroll"}, fmt.Errorf("input not available on this platform")
	}

	direction := StringParam(params, "direction", "")
	amount := IntParam(params, "amount", 3)
	x := IntParam(params, "x", 0)
	y := IntParam(params, "y", 0)
	id := IntParam(params, "id", 0)
	text := StringParam(params, "text", "")
	ref := StringParam(params, "ref", "")
	roles := StringParam(params, "roles", "")
	exact := BoolParam(params, "exact", false)
	scopeID := IntParam(params, "scope-id", 0)

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

	if ref != "" {
		elem, _, err := resolveElementByRef(provider, app, window, 0, 0, ref)
		if err != nil {
			return StepResult{Action: "scroll"}, err
		}
		x = elem.Bounds[0] + elem.Bounds[2]/2
		y = elem.Bounds[1] + elem.Bounds[3]/2
	} else if text != "" {
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

func ExecuteWait(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.Reader == nil {
		return StepResult{Action: "wait"}, fmt.Errorf("reader not available on this platform")
	}

	forText := StringParam(params, "for-text", "")
	forRole := StringParam(params, "for-role", "")
	forID := IntParam(params, "for-id", 0)
	gone := BoolParam(params, "gone", false)
	timeoutSec := IntParam(params, "timeout", 30)
	intervalMs := IntParam(params, "interval", 500)
	pid := IntParam(params, "pid", 0)
	windowID := IntParam(params, "window-id", 0)

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

func ExecuteFocus(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.WindowManager == nil {
		return StepResult{Action: "focus"}, fmt.Errorf("window management not available on this platform")
	}

	windowID := IntParam(params, "window-id", 0)
	pid := IntParam(params, "pid", 0)
	newDocument := BoolParam(params, "new-document", false)

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

	if newDocument {
		if provider.Inputter == nil {
			return StepResult{Action: "focus"}, fmt.Errorf("input simulation not available (required for new-document)")
		}
		time.Sleep(300 * time.Millisecond)
		if err := provider.Inputter.KeyCombo([]string{"escape"}); err != nil {
			return StepResult{Action: "focus"}, fmt.Errorf("failed to dismiss dialog: %w", err)
		}
		time.Sleep(200 * time.Millisecond)
		if err := provider.Inputter.KeyCombo([]string{"cmd", "n"}); err != nil {
			return StepResult{Action: "focus"}, fmt.Errorf("failed to create new document: %w", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	return StepResult{Action: "focus"}, nil
}

func ExecuteRead(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.Reader == nil {
		return StepResult{Action: "read"}, fmt.Errorf("reader not available on this platform")
	}

	formatStr := StringParam(params, "format", "agent")
	pid := IntParam(params, "pid", 0)
	windowID := IntParam(params, "window-id", 0)

	readOpts := platform.ReadOptions{
		App:      app,
		Window:   window,
		PID:      pid,
		WindowID: windowID,
		Depth:    IntParam(params, "depth", 0),
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

func ExecuteAssert(provider *platform.Provider, params map[string]interface{}, app, window string) (StepResult, error) {
	if provider.Reader == nil {
		return StepResult{Action: "assert"}, fmt.Errorf("reader not available on this platform")
	}

	text := StringParam(params, "text", "")
	id := IntParam(params, "id", 0)
	roles := StringParam(params, "roles", "")
	exact := BoolParam(params, "exact", false)
	scopeID := IntParam(params, "scope-id", 0)
	windowID := IntParam(params, "window-id", 0)
	pid := IntParam(params, "pid", 0)
	value := StringParam(params, "value", "")
	valueContains := StringParam(params, "value-contains", "")
	checked := BoolParam(params, "checked", false)
	unchecked := BoolParam(params, "unchecked", false)
	disabled := BoolParam(params, "disabled", false)
	enabled := BoolParam(params, "enabled", false)
	isFocused := BoolParam(params, "focused", false)
	gone := BoolParam(params, "gone", false)
	timeoutSec := IntParam(params, "timeout", 0)
	intervalMs := IntParam(params, "interval", 500)

	if text == "" && id == 0 {
		return StepResult{Action: "assert"}, fmt.Errorf("specify text or id to target an element")
	}

	_, hasValue := params["value"]

	opts := assertOptions{
		provider:      provider,
		appName:       app,
		window:        window,
		windowID:      windowID,
		pid:           pid,
		text:          text,
		roles:         roles,
		exact:         exact,
		scopeID:       scopeID,
		id:            id,
		value:         value,
		hasValueCheck: hasValue,
		valueContains: valueContains,
		checked:       checked,
		unchecked:     unchecked,
		disabled:      disabled,
		enabled:       enabled,
		isFocused:     isFocused,
		gone:          gone,
	}

	if timeoutSec > 0 {
		timeout := time.Duration(timeoutSec) * time.Second
		interval := time.Duration(intervalMs) * time.Millisecond
		deadline := time.Now().Add(timeout)
		start := time.Now()

		for {
			result := checkAssert(opts)
			if result.Pass {
				elapsed := time.Since(start)
				return StepResult{Action: "assert", Target: result.Element, Elapsed: fmt.Sprintf("%.1fs", elapsed.Seconds())}, nil
			}
			if time.Now().After(deadline) {
				return StepResult{Action: "assert"}, fmt.Errorf("assert failed: %s", result.Error)
			}
			time.Sleep(interval)
		}
	}

	result := checkAssert(opts)
	if !result.Pass {
		return StepResult{Action: "assert"}, fmt.Errorf("assert failed: %s", result.Error)
	}
	return StepResult{Action: "assert", Target: result.Element}, nil
}

func ExecuteSleep(params map[string]interface{}) (StepResult, error) {
	ms := IntParam(params, "ms", 0)
	if ms <= 0 {
		return StepResult{Action: "sleep"}, fmt.Errorf("ms must be > 0")
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return StepResult{Action: "sleep", Elapsed: fmt.Sprintf("%dms", ms)}, nil
}

// Parameter extraction helpers for step maps

func StringParam(params map[string]interface{}, key, defaultVal string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		// Handle numeric values that YAML may parse as int/float
		return fmt.Sprintf("%v", v)
	}
	return defaultVal
}

func IntParam(params map[string]interface{}, key string, defaultVal int) int {
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

func BoolParam(params map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}

func ExecuteOpen(params map[string]interface{}) (StepResult, error) {
	urlStr := StringParam(params, "url", "")
	fileStr := StringParam(params, "file", "")
	app := StringParam(params, "app", "")

	if urlStr == "" && fileStr == "" && app == "" {
		return StepResult{Action: "open"}, fmt.Errorf("specify url, file, or app")
	}

	var openArgs []string
	if app != "" {
		openArgs = append(openArgs, "-a", app)
	}
	if urlStr != "" {
		openArgs = append(openArgs, urlStr)
	} else if fileStr != "" {
		openArgs = append(openArgs, fileStr)
	}

	openExec := exec.Command("open", openArgs...)
	if out, err := openExec.CombinedOutput(); err != nil {
		return StepResult{Action: "open"}, fmt.Errorf("open failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	return StepResult{Action: "open"}, nil
}
