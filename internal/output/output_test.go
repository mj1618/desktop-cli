package output

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/mj1618/desktop-cli/internal/model"
	"gopkg.in/yaml.v3"
)

func TestPrintYAML(t *testing.T) {
	result := ReadResult{
		App:    "Safari",
		PID:    1234,
		Window: "GitHub",
		TS:     1707500000,
		Elements: []model.Element{
			{ID: 1, Role: "btn", Title: "OK", Bounds: [4]int{10, 20, 100, 30}},
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintYAML(result)
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// YAML output should be multi-line
	if bytes.Count([]byte(output), []byte("\n")) <= 1 {
		t.Errorf("YAML output should be multi-line, got:\n%s", output)
	}

	// Verify it's valid YAML
	var decoded ReadResult
	if err := yaml.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("output is not valid YAML: %v", err)
	}
	if decoded.App != "Safari" {
		t.Errorf("app: got %q, want %q", decoded.App, "Safari")
	}
	if len(decoded.Elements) != 1 {
		t.Errorf("elements: got %d, want 1", len(decoded.Elements))
	}
}

func TestPrintYAML_NoQuotedY(t *testing.T) {
	// Regression test: "y" should not be quoted in YAML output.
	// gopkg.in/yaml.v3 quotes "y" because it's a YAML 1.1 boolean keyword.
	type ClickResult struct {
		OK     bool   `yaml:"ok"     json:"ok"`
		Action string `yaml:"action" json:"action"`
		X      int    `yaml:"x"      json:"x"`
		Y      int    `yaml:"y"      json:"y"`
		Button string `yaml:"button" json:"button"`
	}
	result := ClickResult{OK: true, Action: "click", X: 100, Y: 200, Button: "left"}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintYAML(result)
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if bytes.Contains([]byte(output), []byte(`"y":`)) {
		t.Errorf("YAML output should not quote 'y' key, got:\n%s", output)
	}
	if !bytes.Contains([]byte(output), []byte("y: 200")) {
		t.Errorf("YAML output should contain unquoted 'y: 200', got:\n%s", output)
	}

	// Verify it's still valid YAML
	var decoded ClickResult
	if err := yaml.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("output is not valid YAML: %v", err)
	}
	if decoded.Y != 200 {
		t.Errorf("y: got %d, want 200", decoded.Y)
	}
}

// captureStdout runs fn and returns whatever it wrote to os.Stdout.
func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := fn()
	w.Close()
	os.Stdout = old
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func boolPtr(b bool) *bool { return &b }

func TestPrintAgent_FiltersToInteractive(t *testing.T) {
	result := ReadFlatResult{
		PID: 100,
		TS:  123,
		Elements: []model.FlatElement{
			{ID: 1, Role: "window", Title: "My App", Bounds: [4]int{0, 0, 800, 600}, Actions: []string{"raise"}},
			{ID: 2, Role: "group", Bounds: [4]int{0, 0, 800, 50}, Actions: []string{"showmenu"}},
			{ID: 3, Role: "btn", Description: "Save", Bounds: [4]int{10, 10, 80, 30}, Actions: []string{"press", "showmenu"}},
			{ID: 4, Role: "txt", Title: "Hello", Bounds: [4]int{100, 10, 50, 20}, Actions: []string{"showmenu"}},
			{ID: 5, Role: "lnk", Description: "Click me", Bounds: [4]int{200, 10, 60, 20}, Actions: []string{"press"}},
			{ID: 6, Role: "img", Bounds: [4]int{300, 10, 20, 20}, Actions: []string{"showmenu"}},
		},
	}

	out := captureStdout(t, func() error { return PrintAgent(result) })

	// Should contain interactive elements
	if !strings.Contains(out, `[3] btn "Save"`) {
		t.Errorf("should contain btn, got:\n%s", out)
	}
	if !strings.Contains(out, `[5] lnk "Click me"`) {
		t.Errorf("should contain lnk, got:\n%s", out)
	}
	// Should NOT contain non-interactive elements
	if strings.Contains(out, "[1]") {
		t.Errorf("should not contain window element, got:\n%s", out)
	}
	if strings.Contains(out, "[2]") {
		t.Errorf("should not contain group element, got:\n%s", out)
	}
	if strings.Contains(out, "[4]") {
		t.Errorf("should not contain txt element, got:\n%s", out)
	}
	if strings.Contains(out, "[6]") {
		t.Errorf("should not contain img element, got:\n%s", out)
	}
}

func TestPrintAgent_LineFormat(t *testing.T) {
	result := ReadFlatResult{
		TS: 123,
		Elements: []model.FlatElement{
			{ID: 42, Role: "btn", Description: "OK", Bounds: [4]int{10, 20, 100, 30}, Actions: []string{"press"}},
		},
	}

	out := captureStdout(t, func() error { return PrintAgent(result) })

	expected := `[42] btn "OK" (10,20,100,30)`
	if !strings.Contains(out, expected) {
		t.Errorf("expected %q in output, got:\n%s", expected, out)
	}
}

func TestPrintAgent_Annotations(t *testing.T) {
	result := ReadFlatResult{
		TS: 123,
		Elements: []model.FlatElement{
			{ID: 1, Role: "btn", Description: "Go", Bounds: [4]int{0, 0, 10, 10}, Enabled: boolPtr(false), Actions: []string{"press"}},
			{ID: 2, Role: "radio", Description: "Opt A", Bounds: [4]int{0, 0, 10, 10}, Selected: true, Actions: []string{"press"}},
			{ID: 3, Role: "input", Description: "Name", Bounds: [4]int{0, 0, 10, 10}, Value: "hello", Focused: true, Actions: []string{"press"}},
			{ID: 4, Role: "chk", Description: "Agree", Bounds: [4]int{0, 0, 10, 10}, Value: "1", Actions: []string{"press"}},
			{ID: 5, Role: "chk", Description: "Other", Bounds: [4]int{0, 0, 10, 10}, Value: "0", Actions: []string{"press"}},
		},
	}

	out := captureStdout(t, func() error { return PrintAgent(result) })

	if !strings.Contains(out, `[1] btn "Go" (0,0,10,10) disabled`) {
		t.Errorf("should show disabled, got:\n%s", out)
	}
	if !strings.Contains(out, `[2] radio "Opt A" (0,0,10,10) selected`) {
		t.Errorf("should show selected, got:\n%s", out)
	}
	if !strings.Contains(out, `focused`) {
		t.Errorf("should show focused, got:\n%s", out)
	}
	if !strings.Contains(out, `val="hello"`) {
		t.Errorf("should show val for input, got:\n%s", out)
	}
	if !strings.Contains(out, `[4] chk "Agree" (0,0,10,10) checked`) {
		t.Errorf("should show checked, got:\n%s", out)
	}
	if !strings.Contains(out, `[5] chk "Other" (0,0,10,10) unchecked`) {
		t.Errorf("should show unchecked, got:\n%s", out)
	}
}

func TestPrintAgent_LabelTruncation(t *testing.T) {
	longLabel := strings.Repeat("a", 100)
	result := ReadFlatResult{
		TS: 123,
		Elements: []model.FlatElement{
			{ID: 1, Role: "btn", Title: longLabel, Bounds: [4]int{0, 0, 10, 10}, Actions: []string{"press"}},
		},
	}

	out := captureStdout(t, func() error { return PrintAgent(result) })

	// Label should be truncated to 80 chars (77 + "...")
	if strings.Contains(out, longLabel) {
		t.Errorf("long label should be truncated, got:\n%s", out)
	}
	if !strings.Contains(out, "...") {
		t.Errorf("truncated label should end with ..., got:\n%s", out)
	}
}

func TestPrintAgent_Header(t *testing.T) {
	result := ReadFlatResult{
		App:    "Chrome",
		PID:    1234,
		Window: "My Page",
		TS:     123,
		Elements: []model.FlatElement{
			{ID: 1, Role: "btn", Description: "OK", Bounds: [4]int{0, 0, 10, 10}, Actions: []string{"press"}},
		},
	}

	out := captureStdout(t, func() error { return PrintAgent(result) })

	if !strings.Contains(out, "# My Page - Chrome (pid: 1234)") {
		t.Errorf("header should include window, app, pid, got:\n%s", out)
	}
}

func TestPrintAgent_HeaderFromWindowElement(t *testing.T) {
	result := ReadFlatResult{
		PID: 5678,
		TS:  123,
		Elements: []model.FlatElement{
			{ID: 1, Role: "window", Title: "Gmail - Chrome", Bounds: [4]int{0, 0, 800, 600}},
			{ID: 2, Role: "btn", Description: "Send", Bounds: [4]int{0, 0, 10, 10}, Actions: []string{"press"}},
		},
	}

	out := captureStdout(t, func() error { return PrintAgent(result) })

	if !strings.Contains(out, "# Gmail - Chrome (pid: 5678)") {
		t.Errorf("header should use window element title, got:\n%s", out)
	}
}

func TestPrintAgent_TreeInput(t *testing.T) {
	result := ReadResult{
		PID: 100,
		TS:  123,
		Elements: []model.Element{
			{
				ID: 1, Role: "window", Title: "App", Bounds: [4]int{0, 0, 800, 600},
				Children: []model.Element{
					{ID: 2, Role: "btn", Description: "Click", Bounds: [4]int{10, 10, 80, 30}, Actions: []string{"press"}},
					{ID: 3, Role: "txt", Title: "Label", Bounds: [4]int{100, 10, 50, 20}},
				},
			},
		},
	}

	out := captureStdout(t, func() error { return PrintAgent(result) })

	if !strings.Contains(out, `[2|click] btn "Click"`) {
		t.Errorf("should contain btn from tree, got:\n%s", out)
	}
	if strings.Contains(out, "[3]") {
		t.Errorf("should not contain txt from tree, got:\n%s", out)
	}
}

func TestPrintAgent_DisplayText(t *testing.T) {
	result := ReadFlatResult{
		TS: 123,
		Elements: []model.FlatElement{
			{ID: 1, Role: "btn", Description: "Send", Bounds: [4]int{0, 0, 10, 10}, Actions: []string{"press"}},
			// txt with value should be included as display
			{ID: 2, Role: "txt", Title: "Formula", Value: "347×29+156", Bounds: [4]int{100, 10, 120, 26}},
			// txt with value but no title — should use value as label
			{ID: 3, Role: "txt", Value: "10219", Bounds: [4]int{100, 40, 80, 36}},
			// txt without value should NOT be included
			{ID: 4, Role: "txt", Title: "Static Label", Bounds: [4]int{200, 10, 50, 20}},
		},
	}

	out := captureStdout(t, func() error { return PrintAgent(result) })

	// Display text with title and value
	if !strings.Contains(out, `[2] txt "Formula"`) {
		t.Errorf("should contain display txt with title, got:\n%s", out)
	}
	if !strings.Contains(out, "display") {
		t.Errorf("should have display annotation, got:\n%s", out)
	}
	if !strings.Contains(out, `val="347×29+156"`) {
		t.Errorf("should show value for display txt with title, got:\n%s", out)
	}

	// Display text with value only (value used as label)
	if !strings.Contains(out, `[3] txt "10219"`) {
		t.Errorf("should contain display txt using value as label, got:\n%s", out)
	}

	// Static txt without value should be excluded
	if strings.Contains(out, "[4]") {
		t.Errorf("should not contain txt without value, got:\n%s", out)
	}
}

func TestPrintAgent_FiltersZeroDimensionElements(t *testing.T) {
	result := ReadFlatResult{
		TS: 123,
		Elements: []model.FlatElement{
			// Normal interactive element — should be included
			{ID: 1, Role: "btn", Description: "OK", Bounds: [4]int{10, 20, 100, 30}, Actions: []string{"press"}},
			// Zero height — should be excluded
			{ID: 2, Role: "btn", Description: "Hidden", Bounds: [4]int{100, 200, 80, 0}, Actions: []string{"press"}},
			// Zero width — should be excluded
			{ID: 3, Role: "lnk", Description: "Invisible", Bounds: [4]int{200, 300, 0, 20}, Actions: []string{"press"}},
			// Zero both — should be excluded
			{ID: 4, Role: "btn", Description: "Ghost", Bounds: [4]int{50, 50, 0, 0}, Actions: []string{"press"}},
			// Display text with zero height — should be excluded
			{ID: 5, Role: "txt", Title: "Offscreen", Value: "123", Bounds: [4]int{500, 600, 100, 0}},
			// Display text with valid bounds — should be included
			{ID: 6, Role: "txt", Title: "Display", Value: "456", Bounds: [4]int{500, 600, 100, 20}},
		},
	}

	out := captureStdout(t, func() error { return PrintAgent(result) })

	if !strings.Contains(out, `[1] btn "OK"`) {
		t.Errorf("should contain normal btn, got:\n%s", out)
	}
	if strings.Contains(out, "[2]") {
		t.Errorf("should not contain zero-height btn, got:\n%s", out)
	}
	if strings.Contains(out, "[3]") {
		t.Errorf("should not contain zero-width lnk, got:\n%s", out)
	}
	if strings.Contains(out, "[4]") {
		t.Errorf("should not contain zero-dimension btn, got:\n%s", out)
	}
	if strings.Contains(out, "[5]") {
		t.Errorf("should not contain zero-height display txt, got:\n%s", out)
	}
	if !strings.Contains(out, `[6] txt "Display"`) {
		t.Errorf("should contain valid display txt, got:\n%s", out)
	}
}

func TestPrintAgent_FallbackToYAML(t *testing.T) {
	type OtherResult struct {
		OK bool `yaml:"ok"`
	}
	result := OtherResult{OK: true}

	out := captureStdout(t, func() error { return PrintAgent(result) })

	if !strings.Contains(out, "ok: true") {
		t.Errorf("non-read result should fall back to YAML, got:\n%s", out)
	}
}

func TestPrintAgent_TitleOverDescription(t *testing.T) {
	result := ReadFlatResult{
		TS: 123,
		Elements: []model.FlatElement{
			{ID: 1, Role: "btn", Title: "My Title", Description: "My Desc", Bounds: [4]int{0, 0, 10, 10}, Actions: []string{"press"}},
		},
	}

	out := captureStdout(t, func() error { return PrintAgent(result) })

	if !strings.Contains(out, `"My Title"`) {
		t.Errorf("should prefer title over description, got:\n%s", out)
	}
	if strings.Contains(out, `"My Desc"`) {
		t.Errorf("should not show description when title is present, got:\n%s", out)
	}
}

func TestReadResult_OmitEmpty(t *testing.T) {
	result := ReadResult{
		TS:       123,
		Elements: []model.Element{},
	}
	data, err := yaml.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	// App, PID, Window should be omitted when empty/zero
	if _, ok := m["app"]; ok {
		t.Error("empty app should be omitted")
	}
	if _, ok := m["pid"]; ok {
		t.Error("zero pid should be omitted")
	}
	if _, ok := m["window"]; ok {
		t.Error("empty window should be omitted")
	}
	// TS should always be present
	if _, ok := m["ts"]; !ok {
		t.Error("ts should always be present")
	}
}
