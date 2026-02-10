//go:build darwin && cgo

package darwin

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Clipboard implements platform.ClipboardManager using pbcopy/pbpaste.
type Clipboard struct{}

// NewClipboard returns a new Clipboard instance.
func NewClipboard() *Clipboard {
	return &Clipboard{}
}

// GetText reads the current text content from the system clipboard.
func (c *Clipboard) GetText() (string, error) {
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return "", fmt.Errorf("pbpaste: %w", err)
	}
	return string(out), nil
}

// SetText writes text to the system clipboard.
func (c *Clipboard) SetText(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pbcopy: %w", err)
	}
	return nil
}

// Clear empties the system clipboard.
func (c *Clipboard) Clear() error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = bytes.NewReader(nil)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pbcopy: %w", err)
	}
	return nil
}
