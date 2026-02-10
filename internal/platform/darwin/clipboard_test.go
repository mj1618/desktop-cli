//go:build darwin && cgo

package darwin

import (
	"testing"
)

func TestClipboardRoundTrip(t *testing.T) {
	c := NewClipboard()

	// Write text to clipboard and read it back
	text := "hello clipboard test"
	if err := c.SetText(text); err != nil {
		t.Fatalf("SetText: %v", err)
	}

	got, err := c.GetText()
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	if got != text {
		t.Errorf("GetText = %q, want %q", got, text)
	}
}

func TestClipboardUnicode(t *testing.T) {
	c := NewClipboard()

	text := "Hello 🌍 — café ñ 中文"
	if err := c.SetText(text); err != nil {
		t.Fatalf("SetText: %v", err)
	}

	got, err := c.GetText()
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	if got != text {
		t.Errorf("GetText = %q, want %q", got, text)
	}
}

func TestClipboardWhitespace(t *testing.T) {
	c := NewClipboard()

	text := "  line1\n\tline2\n  line3  "
	if err := c.SetText(text); err != nil {
		t.Fatalf("SetText: %v", err)
	}

	got, err := c.GetText()
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	if got != text {
		t.Errorf("GetText = %q, want %q", got, text)
	}
}

func TestClipboardClear(t *testing.T) {
	c := NewClipboard()

	// Write something first
	if err := c.SetText("not empty"); err != nil {
		t.Fatalf("SetText: %v", err)
	}

	// Clear clipboard
	if err := c.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	got, err := c.GetText()
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	if got != "" {
		t.Errorf("after Clear, GetText = %q, want empty string", got)
	}
}

func TestClipboardEmptyString(t *testing.T) {
	c := NewClipboard()

	// Writing empty string should work
	if err := c.SetText(""); err != nil {
		t.Fatalf("SetText empty: %v", err)
	}

	got, err := c.GetText()
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	if got != "" {
		t.Errorf("GetText = %q, want empty string", got)
	}
}
