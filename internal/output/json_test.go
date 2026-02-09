package output

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/mj1618/desktop-cli/internal/model"
)

func TestPrintJSON_Compact(t *testing.T) {
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

	err := PrintJSON(result, false)
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Compact output should be a single line (plus newline from Encode)
	if bytes.Count([]byte(output), []byte("\n")) > 1 {
		t.Errorf("compact output should be single line, got:\n%s", output)
	}

	// Verify it's valid JSON
	var decoded ReadResult
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if decoded.App != "Safari" {
		t.Errorf("app: got %q, want %q", decoded.App, "Safari")
	}
	if len(decoded.Elements) != 1 {
		t.Errorf("elements: got %d, want 1", len(decoded.Elements))
	}
}

func TestPrintJSON_Pretty(t *testing.T) {
	result := ReadResult{
		App: "Test",
		TS:  123,
		Elements: []model.Element{
			{ID: 1, Role: "btn", Bounds: [4]int{0, 0, 10, 10}},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintJSON(result, true)
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Pretty output should have multiple lines
	if bytes.Count([]byte(output), []byte("\n")) <= 1 {
		t.Errorf("pretty output should be multi-line, got:\n%s", output)
	}

	// Verify it's valid JSON
	var decoded ReadResult
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestReadResult_OmitEmpty(t *testing.T) {
	result := ReadResult{
		TS:       123,
		Elements: []model.Element{},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
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
