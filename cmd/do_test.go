package cmd

import "testing"

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
