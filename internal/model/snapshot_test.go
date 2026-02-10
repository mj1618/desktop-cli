package model

import (
	"os"
	"testing"
	"time"
)

func TestElementHash_Stable(t *testing.T) {
	el := FlatElement{ID: 1, Role: "btn", Title: "OK", Path: "window > toolbar"}
	h1 := ElementHash(el)
	h2 := ElementHash(el)
	if h1 != h2 {
		t.Errorf("hash not stable: %s != %s", h1, h2)
	}
}

func TestElementHash_IgnoresID(t *testing.T) {
	el1 := FlatElement{ID: 1, Role: "btn", Title: "OK", Path: "window"}
	el2 := FlatElement{ID: 99, Role: "btn", Title: "OK", Path: "window"}
	if ElementHash(el1) != ElementHash(el2) {
		t.Error("hash should not depend on ID")
	}
}

func TestElementHash_IgnoresValue(t *testing.T) {
	el1 := FlatElement{ID: 1, Role: "input", Title: "Search", Value: "old", Path: "window"}
	el2 := FlatElement{ID: 1, Role: "input", Title: "Search", Value: "new", Path: "window"}
	if ElementHash(el1) != ElementHash(el2) {
		t.Error("hash should not depend on value (value is mutable)")
	}
}

func TestElementHash_DiffersByRole(t *testing.T) {
	el1 := FlatElement{Role: "btn", Title: "OK", Path: "window"}
	el2 := FlatElement{Role: "lnk", Title: "OK", Path: "window"}
	if ElementHash(el1) == ElementHash(el2) {
		t.Error("different roles should produce different hashes")
	}
}

func TestElementHash_DiffersByPath(t *testing.T) {
	el1 := FlatElement{Role: "btn", Title: "OK", Path: "window > toolbar"}
	el2 := FlatElement{Role: "btn", Title: "OK", Path: "window > footer"}
	if ElementHash(el1) == ElementHash(el2) {
		t.Error("different paths should produce different hashes")
	}
}

func TestDiffElementsByHash_NoChanges(t *testing.T) {
	elements := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Bounds: [4]int{10, 20, 100, 30}, Path: "window"},
	}
	diff := DiffElementsByHash(elements, elements)
	if len(diff.Added) != 0 {
		t.Errorf("expected no added, got %d", len(diff.Added))
	}
	if len(diff.Removed) != 0 {
		t.Errorf("expected no removed, got %d", len(diff.Removed))
	}
	if len(diff.Changed) != 0 {
		t.Errorf("expected no changed, got %d", len(diff.Changed))
	}
	if diff.UnchangedCount != 1 {
		t.Errorf("expected 1 unchanged, got %d", diff.UnchangedCount)
	}
}

func TestDiffElementsByHash_Added(t *testing.T) {
	prev := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Path: "window"},
	}
	curr := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Path: "window"},
		{ID: 2, Role: "btn", Title: "Cancel", Path: "window"},
	}
	diff := DiffElementsByHash(prev, curr)
	if len(diff.Added) != 1 {
		t.Fatalf("expected 1 added, got %d", len(diff.Added))
	}
	if diff.Added[0].Title != "Cancel" {
		t.Errorf("expected Cancel, got %s", diff.Added[0].Title)
	}
	if diff.UnchangedCount != 1 {
		t.Errorf("expected 1 unchanged, got %d", diff.UnchangedCount)
	}
}

func TestDiffElementsByHash_Removed(t *testing.T) {
	prev := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Path: "window"},
		{ID: 2, Role: "btn", Title: "Loading...", Path: "window"},
	}
	curr := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Path: "window"},
	}
	diff := DiffElementsByHash(prev, curr)
	if len(diff.Removed) != 1 {
		t.Fatalf("expected 1 removed, got %d", len(diff.Removed))
	}
	if diff.Removed[0].Title != "Loading..." {
		t.Errorf("expected Loading..., got %s", diff.Removed[0].Title)
	}
}

func TestDiffElementsByHash_Changed(t *testing.T) {
	prev := []FlatElement{
		{ID: 1, Role: "input", Title: "Search", Value: "", Path: "window"},
	}
	curr := []FlatElement{
		{ID: 1, Role: "input", Title: "Search", Value: "hello", Path: "window"},
	}
	diff := DiffElementsByHash(prev, curr)
	if len(diff.Changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(diff.Changed))
	}
	if diff.Changed[0].Changes["v"][1] != "hello" {
		t.Errorf("expected new value 'hello', got %s", diff.Changed[0].Changes["v"][1])
	}
}

func TestDiffElementsByHash_IDShift(t *testing.T) {
	// Simulate an element being inserted at the beginning, shifting all IDs
	prev := []FlatElement{
		{ID: 1, Role: "btn", Title: "A", Path: "window"},
		{ID: 2, Role: "btn", Title: "B", Path: "window"},
	}
	curr := []FlatElement{
		{ID: 1, Role: "btn", Title: "New", Path: "window"},
		{ID: 2, Role: "btn", Title: "A", Path: "window"},
		{ID: 3, Role: "btn", Title: "B", Path: "window"},
	}
	diff := DiffElementsByHash(prev, curr)
	if len(diff.Added) != 1 {
		t.Fatalf("expected 1 added (New), got %d", len(diff.Added))
	}
	if diff.Added[0].Title != "New" {
		t.Errorf("expected New, got %s", diff.Added[0].Title)
	}
	if diff.UnchangedCount != 2 {
		t.Errorf("expected 2 unchanged (A, B), got %d", diff.UnchangedCount)
	}
	if len(diff.Removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(diff.Removed))
	}
}

func TestDiffElementsByHash_Empty(t *testing.T) {
	diff := DiffElementsByHash(nil, nil)
	if len(diff.Added) != 0 || len(diff.Removed) != 0 || len(diff.Changed) != 0 {
		t.Error("expected empty diff for nil inputs")
	}
	if diff.UnchangedCount != 0 {
		t.Errorf("expected 0 unchanged, got %d", diff.UnchangedCount)
	}
}

func TestDiffElementsByHash_AllNew(t *testing.T) {
	curr := []FlatElement{
		{ID: 1, Role: "btn", Title: "A", Path: "window"},
		{ID: 2, Role: "txt", Title: "B", Path: "window"},
	}
	diff := DiffElementsByHash(nil, curr)
	if len(diff.Added) != 2 {
		t.Errorf("expected 2 added, got %d", len(diff.Added))
	}
}

func TestDiffElementsByHash_AllRemoved(t *testing.T) {
	prev := []FlatElement{
		{ID: 1, Role: "btn", Title: "A", Path: "window"},
		{ID: 2, Role: "txt", Title: "B", Path: "window"},
	}
	diff := DiffElementsByHash(prev, nil)
	if len(diff.Removed) != 2 {
		t.Errorf("expected 2 removed, got %d", len(diff.Removed))
	}
}

func TestDiffElementsByHash_BoundsChange(t *testing.T) {
	prev := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Bounds: [4]int{10, 20, 100, 30}, Path: "window"},
	}
	curr := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Bounds: [4]int{10, 20, 200, 30}, Path: "window"},
	}
	diff := DiffElementsByHash(prev, curr)
	if len(diff.Changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(diff.Changed))
	}
	if _, ok := diff.Changed[0].Changes["b"]; !ok {
		t.Error("expected bounds change")
	}
}

func TestSaveLoadSnapshot(t *testing.T) {
	app := "test-snapshot-app"
	ts := time.Now().Unix()
	elements := []FlatElement{
		{ID: 1, Role: "btn", Title: "OK", Bounds: [4]int{10, 20, 100, 30}, Path: "window"},
		{ID: 2, Role: "input", Title: "Search", Value: "hello", Path: "window"},
	}

	if err := SaveSnapshot(app, ts, elements); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	defer os.Remove(snapshotPath(app, ts))

	loaded, err := LoadSnapshot(app, ts)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(loaded))
	}
	if loaded[0].Title != "OK" {
		t.Errorf("expected OK, got %s", loaded[0].Title)
	}
	if loaded[1].Value != "hello" {
		t.Errorf("expected hello, got %s", loaded[1].Value)
	}
}

func TestLoadSnapshot_NotFound(t *testing.T) {
	_, err := LoadSnapshot("nonexistent-app", 0)
	if err == nil {
		t.Error("expected error for missing snapshot")
	}
}
