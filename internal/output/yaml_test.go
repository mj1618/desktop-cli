package output

import (
	"bytes"
	"os"
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
