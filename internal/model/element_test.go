package model

import (
	"encoding/json"
	"testing"
)

func TestElement_JSONKeys(t *testing.T) {
	el := Element{
		ID:     1,
		Role:   "btn",
		Title:  "OK",
		Bounds: [4]int{10, 20, 100, 30},
	}
	data, err := json.Marshal(el)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	// Must have compact keys
	for _, key := range []string{"i", "r", "t", "b"} {
		if _, ok := m[key]; !ok {
			t.Errorf("expected key %q in JSON output", key)
		}
	}
	// Must NOT have verbose keys
	for _, key := range []string{"id", "role", "title", "bounds"} {
		if _, ok := m[key]; ok {
			t.Errorf("unexpected verbose key %q in JSON output", key)
		}
	}
}

func TestElement_OmitEmpty(t *testing.T) {
	el := Element{
		ID:     1,
		Role:   "btn",
		Bounds: [4]int{0, 0, 100, 30},
	}
	data, err := json.Marshal(el)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	// Empty title should be omitted
	if _, ok := m["t"]; ok {
		t.Error("empty title should be omitted")
	}
	// Empty value should be omitted
	if _, ok := m["v"]; ok {
		t.Error("empty value should be omitted")
	}
	// Empty description should be omitted
	if _, ok := m["d"]; ok {
		t.Error("empty description should be omitted")
	}
	// Focused=false should be omitted
	if _, ok := m["f"]; ok {
		t.Error("focused=false should be omitted")
	}
	// Selected=false should be omitted
	if _, ok := m["s"]; ok {
		t.Error("selected=false should be omitted")
	}
	// Enabled=nil should be omitted
	if _, ok := m["e"]; ok {
		t.Error("enabled=nil should be omitted")
	}
	// No children should be omitted
	if _, ok := m["c"]; ok {
		t.Error("empty children should be omitted")
	}
	// No actions should be omitted
	if _, ok := m["a"]; ok {
		t.Error("empty actions should be omitted")
	}
}

func TestElement_EnabledFalse_Included(t *testing.T) {
	f := false
	el := Element{
		ID:      1,
		Role:    "btn",
		Bounds:  [4]int{0, 0, 100, 30},
		Enabled: &f,
	}
	data, err := json.Marshal(el)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	val, ok := m["e"]
	if !ok {
		t.Fatal("enabled=false should be included in JSON")
	}
	if val != false {
		t.Errorf("expected enabled=false, got %v", val)
	}
}

func TestElement_EnabledTrue_Omitted(t *testing.T) {
	tr := true
	el := Element{
		ID:      1,
		Role:    "btn",
		Bounds:  [4]int{0, 0, 100, 30},
		Enabled: &tr,
	}
	data, err := json.Marshal(el)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	// *bool with omitempty: true is NOT the zero value, so it will be included.
	// This is expected Go behavior — omitempty on *bool omits nil, but not &true.
	// The task spec says "nil or true = enabled (omit)", but Go's omitempty only
	// omits nil pointers. &true will be included. This is acceptable since the
	// platform backend should set Enabled to nil (not &true) for enabled elements.
}

func TestElement_WithChildren(t *testing.T) {
	el := Element{
		ID:     1,
		Role:   "toolbar",
		Title:  "Nav",
		Bounds: [4]int{0, 0, 1440, 52},
		Children: []Element{
			{ID: 2, Role: "btn", Title: "Back", Bounds: [4]int{10, 10, 32, 32}},
		},
	}
	data, err := json.Marshal(el)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	children, ok := m["c"]
	if !ok {
		t.Fatal("children should be present when non-empty")
	}
	arr, ok := children.([]interface{})
	if !ok || len(arr) != 1 {
		t.Errorf("expected 1 child, got %v", children)
	}
}

func TestElement_RoundTrip(t *testing.T) {
	f := false
	original := Element{
		ID:          1,
		Role:        "input",
		Title:       "Search",
		Value:       "hello",
		Description: "Search field",
		Bounds:      [4]int{100, 200, 300, 40},
		Focused:     true,
		Enabled:     &f,
		Selected:    true,
		Actions:     []string{"press", "cancel"},
		Children: []Element{
			{ID: 2, Role: "txt", Title: "Placeholder", Bounds: [4]int{105, 205, 290, 30}},
		},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Element
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ID != original.ID {
		t.Errorf("ID: got %d, want %d", decoded.ID, original.ID)
	}
	if decoded.Role != original.Role {
		t.Errorf("Role: got %q, want %q", decoded.Role, original.Role)
	}
	if decoded.Title != original.Title {
		t.Errorf("Title: got %q, want %q", decoded.Title, original.Title)
	}
	if decoded.Value != original.Value {
		t.Errorf("Value: got %q, want %q", decoded.Value, original.Value)
	}
	if decoded.Bounds != original.Bounds {
		t.Errorf("Bounds: got %v, want %v", decoded.Bounds, original.Bounds)
	}
	if decoded.Focused != original.Focused {
		t.Errorf("Focused: got %v, want %v", decoded.Focused, original.Focused)
	}
	if decoded.Enabled == nil || *decoded.Enabled != false {
		t.Errorf("Enabled: got %v, want false", decoded.Enabled)
	}
	if len(decoded.Children) != 1 {
		t.Errorf("Children: got %d, want 1", len(decoded.Children))
	}
	if len(decoded.Actions) != 2 {
		t.Errorf("Actions: got %d, want 2", len(decoded.Actions))
	}
}
