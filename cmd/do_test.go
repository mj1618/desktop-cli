package cmd

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// computeDoResult replicates the batch result logic from runDo to test it
// independently of the platform provider.
func computeDoResult(stepResults []StepResult, stopOnError bool) (allOK bool, completed int) {
	completed = 0
	hasFailure := false
	var lastErr string

	for _, r := range stepResults {
		if !r.OK {
			hasFailure = true
			if stopOnError {
				lastErr = r.Error
				break
			}
		} else {
			completed++
		}
	}

	_ = lastErr // used for Error field in real code
	if !hasFailure {
		completed = len(stepResults)
	}
	allOK = !hasFailure
	return
}

func TestDoResult_AllSuccess(t *testing.T) {
	steps := []StepResult{
		{Step: 1, OK: true, Action: "sleep"},
		{Step: 2, OK: true, Action: "sleep"},
		{Step: 3, OK: true, Action: "sleep"},
	}

	ok, completed := computeDoResult(steps, true)
	if !ok {
		t.Error("expected ok=true when all steps succeed")
	}
	if completed != 3 {
		t.Errorf("expected completed=3, got %d", completed)
	}
}

func TestDoResult_StopOnError_FailAtStep2(t *testing.T) {
	steps := []StepResult{
		{Step: 1, OK: true, Action: "sleep"},
		{Step: 2, OK: false, Action: "unknown-cmd", Error: "unknown step type"},
	}

	ok, completed := computeDoResult(steps, true)
	if ok {
		t.Error("expected ok=false when a step fails with stop-on-error=true")
	}
	if completed != 1 {
		t.Errorf("expected completed=1, got %d", completed)
	}
}

func TestDoResult_ContinueOnError_FailAtStep2(t *testing.T) {
	// This is the exact scenario from the bug report:
	// 3 steps, step 2 fails, stop-on-error=false
	steps := []StepResult{
		{Step: 1, OK: true, Action: "sleep"},
		{Step: 2, OK: false, Action: "unknown-cmd", Error: "unknown step type"},
		{Step: 3, OK: true, Action: "sleep"},
	}

	ok, completed := computeDoResult(steps, false)
	if ok {
		t.Error("expected ok=false when a step fails with stop-on-error=false")
	}
	if completed != 2 {
		t.Errorf("expected completed=2 (only successful steps), got %d", completed)
	}
}

func TestDoResult_ContinueOnError_AllFail(t *testing.T) {
	steps := []StepResult{
		{Step: 1, OK: false, Action: "bad1", Error: "err1"},
		{Step: 2, OK: false, Action: "bad2", Error: "err2"},
	}

	ok, completed := computeDoResult(steps, false)
	if ok {
		t.Error("expected ok=false when all steps fail")
	}
	if completed != 0 {
		t.Errorf("expected completed=0, got %d", completed)
	}
}

func TestDoResult_ContinueOnError_MultipleFails(t *testing.T) {
	steps := []StepResult{
		{Step: 1, OK: true, Action: "sleep"},
		{Step: 2, OK: false, Action: "bad", Error: "err"},
		{Step: 3, OK: true, Action: "sleep"},
		{Step: 4, OK: false, Action: "bad2", Error: "err2"},
		{Step: 5, OK: true, Action: "sleep"},
	}

	ok, completed := computeDoResult(steps, false)
	if ok {
		t.Error("expected ok=false with multiple failures")
	}
	if completed != 3 {
		t.Errorf("expected completed=3, got %d", completed)
	}
}

// --- Conditional step tests ---

func parseSteps(t *testing.T, yamlData string) []map[string]interface{} {
	t.Helper()
	var rawSteps []map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlData), &rawSteps); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}
	return rawSteps
}

func TestParseRegularStep(t *testing.T) {
	step := map[string]interface{}{
		"click": map[string]interface{}{"text": "Submit"},
	}
	action, params, err := parseRegularStep(step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "click" {
		t.Fatalf("expected action 'click', got %q", action)
	}
	if params["text"] != "Submit" {
		t.Fatalf("expected text 'Submit', got %v", params["text"])
	}
}

func TestParseRegularStep_NilParams(t *testing.T) {
	step := map[string]interface{}{"sleep": nil}
	action, params, err := parseRegularStep(step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "sleep" {
		t.Fatalf("expected action 'sleep', got %q", action)
	}
	if params == nil {
		t.Fatal("expected non-nil params map")
	}
}

func TestParseRegularStep_SkipsThenElse(t *testing.T) {
	step := map[string]interface{}{
		"click": map[string]interface{}{"text": "OK"},
		"then":  []interface{}{},
	}
	action, _, err := parseRegularStep(step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "click" {
		t.Fatalf("expected action 'click', got %q", action)
	}
}

func TestParseRegularStep_MultipleActions(t *testing.T) {
	step := map[string]interface{}{
		"click": map[string]interface{}{"text": "OK"},
		"type":  map[string]interface{}{"text": "hello"},
	}
	_, _, err := parseRegularStep(step)
	if err == nil {
		t.Fatal("expected error for multiple action keys")
	}
}

func TestParseSubsteps_Valid(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{"click": map[string]interface{}{"text": "OK"}},
		map[string]interface{}{"type": map[string]interface{}{"text": "hello"}},
	}
	steps, err := parseSubsteps(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
}

func TestParseSubsteps_Nil(t *testing.T) {
	steps, err := parseSubsteps(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if steps != nil {
		t.Fatalf("expected nil, got %v", steps)
	}
}

func TestParseSubsteps_InvalidType(t *testing.T) {
	_, err := parseSubsteps("not an array")
	if err == nil {
		t.Fatal("expected error for non-array input")
	}
}

func TestYAMLParseConditionalSteps(t *testing.T) {
	rawSteps := parseSteps(t, `
- if-exists: { text: "Accept Cookies" }
  then:
    - click: { text: "Accept Cookies" }
- try:
    - click: { text: "Dismiss" }
- click: { text: "Submit" }
`)
	if len(rawSteps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(rawSteps))
	}
	if _, ok := rawSteps[0]["if-exists"]; !ok {
		t.Fatal("expected step 1 to have 'if-exists' key")
	}
	if _, ok := rawSteps[0]["then"]; !ok {
		t.Fatal("expected step 1 to have 'then' key")
	}
	if _, ok := rawSteps[1]["try"]; !ok {
		t.Fatal("expected step 2 to have 'try' key")
	}
	action, params, err := parseRegularStep(rawSteps[2])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "click" {
		t.Fatalf("expected 'click', got %q", action)
	}
	if StringParam(params, "text", "") != "Submit" {
		t.Fatal("expected text 'Submit'")
	}
}

func TestYAMLParseIfExistsWithElse(t *testing.T) {
	rawSteps := parseSteps(t, `
- if-exists: { text: "Sign In", roles: "btn" }
  then:
    - click: { text: "Sign In" }
    - wait: { for-text: "Dashboard", timeout: 10 }
  else:
    - read: { format: "agent" }
`)
	if len(rawSteps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(rawSteps))
	}
	step := rawSteps[0]
	condParams, ok := step["if-exists"].(map[string]interface{})
	if !ok {
		t.Fatal("expected if-exists to be a map")
	}
	if StringParam(condParams, "text", "") != "Sign In" {
		t.Fatal("expected text 'Sign In'")
	}
	if StringParam(condParams, "roles", "") != "btn" {
		t.Fatal("expected roles 'btn'")
	}
	thenSteps, err := parseSubsteps(step["then"])
	if err != nil {
		t.Fatalf("failed to parse then steps: %v", err)
	}
	if len(thenSteps) != 2 {
		t.Fatalf("expected 2 then steps, got %d", len(thenSteps))
	}
	elseSteps, err := parseSubsteps(step["else"])
	if err != nil {
		t.Fatalf("failed to parse else steps: %v", err)
	}
	if len(elseSteps) != 1 {
		t.Fatalf("expected 1 else step, got %d", len(elseSteps))
	}
}

func TestYAMLParseIfFocused(t *testing.T) {
	rawSteps := parseSteps(t, `
- if-focused: { roles: "input" }
  then:
    - type: { text: "search query", key: "enter" }
  else:
    - click: { text: "Search", roles: "input" }
    - type: { text: "search query", key: "enter" }
`)
	if len(rawSteps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(rawSteps))
	}
	step := rawSteps[0]
	if _, ok := step["if-focused"]; !ok {
		t.Fatal("expected 'if-focused' key")
	}
	thenSteps, err := parseSubsteps(step["then"])
	if err != nil {
		t.Fatalf("failed to parse then steps: %v", err)
	}
	if len(thenSteps) != 1 {
		t.Fatalf("expected 1 then step, got %d", len(thenSteps))
	}
	elseSteps, err := parseSubsteps(step["else"])
	if err != nil {
		t.Fatalf("failed to parse else steps: %v", err)
	}
	if len(elseSteps) != 2 {
		t.Fatalf("expected 2 else steps, got %d", len(elseSteps))
	}
}

func TestYAMLParseTry(t *testing.T) {
	rawSteps := parseSteps(t, `
- try:
    - click: { text: "Dismiss" }
    - wait: { for-text: "Dismiss", gone: true, timeout: 2 }
`)
	if len(rawSteps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(rawSteps))
	}
	trySteps, err := parseSubsteps(rawSteps[0]["try"])
	if err != nil {
		t.Fatalf("failed to parse try steps: %v", err)
	}
	if len(trySteps) != 2 {
		t.Fatalf("expected 2 try substeps, got %d", len(trySteps))
	}
}

func TestYAMLParseNestedConditionals(t *testing.T) {
	rawSteps := parseSteps(t, `
- try:
    - if-exists: { text: "Cookie Banner" }
      then:
        - click: { text: "Accept" }
    - click: { text: "Continue" }
`)
	if len(rawSteps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(rawSteps))
	}
	trySteps, err := parseSubsteps(rawSteps[0]["try"])
	if err != nil {
		t.Fatalf("failed to parse try steps: %v", err)
	}
	if len(trySteps) != 2 {
		t.Fatalf("expected 2 try substeps, got %d", len(trySteps))
	}
	if _, ok := trySteps[0]["if-exists"]; !ok {
		t.Fatal("expected first try substep to be if-exists")
	}
	if _, ok := trySteps[0]["then"]; !ok {
		t.Fatal("expected first try substep to have then branch")
	}
	action, _, err := parseRegularStep(trySteps[1])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "click" {
		t.Fatalf("expected 'click', got %q", action)
	}
}

func TestYAMLParseBackwardCompatibility(t *testing.T) {
	rawSteps := parseSteps(t, `
- click: { text: "Full Name" }
- type: { text: "John Doe" }
- type: { key: "tab" }
- sleep: { ms: 100 }
- wait: { for-text: "Thank you", timeout: 10 }
`)
	if len(rawSteps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(rawSteps))
	}
	expectedActions := []string{"click", "type", "type", "sleep", "wait"}
	for i, step := range rawSteps {
		action, _, err := parseRegularStep(step)
		if err != nil {
			t.Fatalf("step %d: unexpected error: %v", i+1, err)
		}
		if action != expectedActions[i] {
			t.Fatalf("step %d: expected %q, got %q", i+1, expectedActions[i], action)
		}
	}
}

func TestDoContextExecuteSteps_SleepOnly(t *testing.T) {
	rawSteps := parseSteps(t, `
- sleep: { ms: 1 }
- sleep: { ms: 1 }
`)
	ctx := &DoContext{StopOnError: true}
	ctx.ExecuteSteps(rawSteps, 0)
	if len(ctx.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ctx.Results))
	}
	for i, r := range ctx.Results {
		if !r.OK {
			t.Fatalf("step %d: expected OK, got error: %s", i+1, r.Error)
		}
		if r.Action != "sleep" {
			t.Fatalf("step %d: expected action 'sleep', got %q", i+1, r.Action)
		}
		if r.Step != i+1 {
			t.Fatalf("step %d: expected step number %d, got %d", i+1, i+1, r.Step)
		}
	}
	if ctx.HasFailure {
		t.Fatal("expected no failure")
	}
}

func TestDoContextTry_AlwaysSucceeds(t *testing.T) {
	rawSteps := parseSteps(t, `
- try:
    - sleep: { ms: -1 }
- sleep: { ms: 1 }
`)
	ctx := &DoContext{StopOnError: true}
	ctx.ExecuteSteps(rawSteps, 0)
	if len(ctx.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ctx.Results))
	}
	tryResult := ctx.Results[0]
	if !tryResult.OK {
		t.Fatalf("try block should be OK, got error: %s", tryResult.Error)
	}
	if tryResult.Action != "try" {
		t.Fatalf("expected action 'try', got %q", tryResult.Action)
	}
	if len(tryResult.Substeps) != 1 {
		t.Fatalf("expected 1 substep, got %d", len(tryResult.Substeps))
	}
	if tryResult.Substeps[0].OK {
		t.Fatal("expected substep to fail (sleep with ms=-1)")
	}
	if !ctx.Results[1].OK {
		t.Fatalf("step after try should succeed, got error: %s", ctx.Results[1].Error)
	}
	if ctx.HasFailure {
		t.Fatal("expected no overall failure (try absorbs errors)")
	}
}

func TestDoContextStopOnError_Conditional(t *testing.T) {
	rawSteps := parseSteps(t, `
- sleep: { ms: -1 }
- sleep: { ms: 1 }
`)
	ctx := &DoContext{StopOnError: true}
	ctx.ExecuteSteps(rawSteps, 0)
	if len(ctx.Results) != 1 {
		t.Fatalf("expected 1 result (stopped on error), got %d", len(ctx.Results))
	}
	if ctx.Results[0].OK {
		t.Fatal("expected first step to fail")
	}
	if !ctx.HasFailure {
		t.Fatal("expected hasFailure to be true")
	}
}

func TestDoContextContinueOnError_Conditional(t *testing.T) {
	rawSteps := parseSteps(t, `
- sleep: { ms: -1 }
- sleep: { ms: 1 }
`)
	ctx := &DoContext{StopOnError: false}
	ctx.ExecuteSteps(rawSteps, 0)
	if len(ctx.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ctx.Results))
	}
	if ctx.Results[0].OK {
		t.Fatal("expected first step to fail")
	}
	if !ctx.Results[1].OK {
		t.Fatalf("expected second step to succeed, got error: %s", ctx.Results[1].Error)
	}
}

func TestDoContextIfExists_NoProvider(t *testing.T) {
	rawSteps := parseSteps(t, `
- if-exists: { text: "Accept" }
  then:
    - sleep: { ms: 1 }
  else:
    - sleep: { ms: 1 }
`)
	ctx := &DoContext{StopOnError: true}
	ctx.ExecuteSteps(rawSteps, 0)
	if len(ctx.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(ctx.Results))
	}
	result := ctx.Results[0]
	if result.Action != "if-exists" {
		t.Fatalf("expected action 'if-exists', got %q", result.Action)
	}
	if !result.OK {
		t.Fatalf("expected OK, got error: %s", result.Error)
	}
	if result.Matched == nil || *result.Matched {
		t.Fatal("expected matched=false (no provider)")
	}
	if result.Branch != "else" {
		t.Fatalf("expected branch 'else', got %q", result.Branch)
	}
	if len(result.Substeps) != 1 {
		t.Fatalf("expected 1 substep, got %d", len(result.Substeps))
	}
	if !result.Substeps[0].OK {
		t.Fatalf("expected else substep to succeed, got error: %s", result.Substeps[0].Error)
	}
}

func TestDoContextIfExists_NoElse(t *testing.T) {
	rawSteps := parseSteps(t, `
- if-exists: { text: "Accept" }
  then:
    - sleep: { ms: 1 }
- sleep: { ms: 1 }
`)
	ctx := &DoContext{StopOnError: true}
	ctx.ExecuteSteps(rawSteps, 0)
	if len(ctx.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ctx.Results))
	}
	ifResult := ctx.Results[0]
	if !ifResult.OK {
		t.Fatalf("expected if-exists to be OK, got error: %s", ifResult.Error)
	}
	if len(ifResult.Substeps) != 0 {
		t.Fatalf("expected 0 substeps (no else), got %d", len(ifResult.Substeps))
	}
	if !ctx.Results[1].OK {
		t.Fatalf("expected second step to succeed, got error: %s", ctx.Results[1].Error)
	}
}

func TestDoContextIfFocused_NoProvider(t *testing.T) {
	rawSteps := parseSteps(t, `
- if-focused: { roles: "input" }
  then:
    - sleep: { ms: 1 }
  else:
    - sleep: { ms: 1 }
`)
	ctx := &DoContext{StopOnError: true}
	ctx.ExecuteSteps(rawSteps, 0)
	if len(ctx.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(ctx.Results))
	}
	result := ctx.Results[0]
	if result.Action != "if-focused" {
		t.Fatalf("expected action 'if-focused', got %q", result.Action)
	}
	if result.Matched == nil || *result.Matched {
		t.Fatal("expected matched=false (no provider)")
	}
	if result.Branch != "else" {
		t.Fatalf("expected branch 'else', got %q", result.Branch)
	}
	if len(result.Substeps) != 1 {
		t.Fatalf("expected 1 substep, got %d", len(result.Substeps))
	}
}

func TestDoContextMixedSteps(t *testing.T) {
	rawSteps := parseSteps(t, `
- try:
    - sleep: { ms: 1 }
- if-exists: { text: "Missing" }
  then:
    - sleep: { ms: 1 }
- sleep: { ms: 1 }
`)
	ctx := &DoContext{StopOnError: true}
	ctx.ExecuteSteps(rawSteps, 0)
	if len(ctx.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(ctx.Results))
	}
	if ctx.Results[0].Action != "try" || !ctx.Results[0].OK {
		t.Fatalf("expected try OK, got action=%q ok=%v", ctx.Results[0].Action, ctx.Results[0].OK)
	}
	if ctx.Results[1].Action != "if-exists" || !ctx.Results[1].OK {
		t.Fatalf("expected if-exists OK, got action=%q ok=%v", ctx.Results[1].Action, ctx.Results[1].OK)
	}
	if ctx.Results[2].Action != "sleep" || !ctx.Results[2].OK {
		t.Fatalf("expected sleep OK, got action=%q ok=%v", ctx.Results[2].Action, ctx.Results[2].OK)
	}
}

func TestDoContextTry_MultipleSubsteps(t *testing.T) {
	rawSteps := parseSteps(t, `
- try:
    - sleep: { ms: 1 }
    - sleep: { ms: -1 }
    - sleep: { ms: 1 }
`)
	ctx := &DoContext{StopOnError: true}
	ctx.ExecuteSteps(rawSteps, 0)
	if len(ctx.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(ctx.Results))
	}
	tryResult := ctx.Results[0]
	if !tryResult.OK {
		t.Fatal("try block should always succeed")
	}
	// try stops within on first error (stopOnError=true within try)
	if len(tryResult.Substeps) != 2 {
		t.Fatalf("expected 2 substeps (stopped at error), got %d", len(tryResult.Substeps))
	}
	if !tryResult.Substeps[0].OK {
		t.Fatal("expected first substep to succeed")
	}
	if tryResult.Substeps[1].OK {
		t.Fatal("expected second substep to fail")
	}
}
