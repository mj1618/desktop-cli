package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// ClipboardReadResult is the output of `clipboard read`.
type ClipboardReadResult struct {
	OK     bool   `yaml:"ok"     json:"ok"`
	Action string `yaml:"action" json:"action"`
	Text   string `yaml:"text"   json:"text"`
}

// ClipboardWriteResult is the output of `clipboard write` and `clipboard clear`.
type ClipboardWriteResult struct {
	OK     bool   `yaml:"ok"     json:"ok"`
	Action string `yaml:"action" json:"action"`
}

// ClipboardGrabResult is the output of `clipboard grab`.
type ClipboardGrabResult struct {
	OK     bool   `yaml:"ok"     json:"ok"`
	Action string `yaml:"action" json:"action"`
	App    string `yaml:"app"    json:"app"`
	Text   string `yaml:"text"   json:"text"`
}

var clipboardCmd = &cobra.Command{
	Use:   "clipboard",
	Short: "Read, write, or clear the system clipboard",
	Long:  "Interact with the system clipboard: read its contents, write text to it, clear it, or grab selected text from an app.",
}

var clipboardReadCmd = &cobra.Command{
	Use:   "read",
	Short: "Read the current clipboard text",
	RunE:  runClipboardRead,
}

var clipboardWriteCmd = &cobra.Command{
	Use:   "write [text]",
	Short: "Write text to the clipboard",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runClipboardWrite,
}

var clipboardClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the clipboard",
	RunE:  runClipboardClear,
}

var clipboardGrabCmd = &cobra.Command{
	Use:   "grab",
	Short: "Select all + copy from an app, then read clipboard",
	Long:  "Focuses the target app, sends Cmd+A then Cmd+C, waits briefly, then reads the clipboard contents.",
	RunE:  runClipboardGrab,
}

func init() {
	rootCmd.AddCommand(clipboardCmd)
	clipboardCmd.AddCommand(clipboardReadCmd)
	clipboardCmd.AddCommand(clipboardWriteCmd)
	clipboardCmd.AddCommand(clipboardClearCmd)
	clipboardCmd.AddCommand(clipboardGrabCmd)

	clipboardWriteCmd.Flags().String("text", "", "Text to write to the clipboard")
	clipboardGrabCmd.Flags().String("app", "", "Application to grab text from (required)")
	clipboardGrabCmd.Flags().String("window", "", "Window title substring")
	clipboardGrabCmd.Flags().Int("window-id", 0, "Window by system ID")
	clipboardGrabCmd.Flags().Int("pid", 0, "Process by PID")
}

func runClipboardRead(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.ClipboardManager == nil {
		return fmt.Errorf("clipboard not supported on this platform")
	}

	text, err := provider.ClipboardManager.GetText()
	if err != nil {
		return err
	}

	return output.Print(ClipboardReadResult{
		OK:     true,
		Action: "clipboard-read",
		Text:   text,
	})
}

func runClipboardWrite(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.ClipboardManager == nil {
		return fmt.Errorf("clipboard not supported on this platform")
	}

	// Get text from positional arg or --text flag
	var text string
	if len(args) > 0 {
		text = args[0]
	}
	if flagText, _ := cmd.Flags().GetString("text"); flagText != "" {
		text = flagText
	}
	if text == "" {
		return fmt.Errorf("specify text as a positional argument or --text flag")
	}

	if err := provider.ClipboardManager.SetText(text); err != nil {
		return err
	}

	return output.Print(ClipboardWriteResult{
		OK:     true,
		Action: "clipboard-write",
	})
}

func runClipboardClear(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.ClipboardManager == nil {
		return fmt.Errorf("clipboard not supported on this platform")
	}

	if err := provider.ClipboardManager.Clear(); err != nil {
		return err
	}

	return output.Print(ClipboardWriteResult{
		OK:     true,
		Action: "clipboard-clear",
	})
}

func runClipboardGrab(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.ClipboardManager == nil {
		return fmt.Errorf("clipboard not supported on this platform")
	}
	if provider.WindowManager == nil {
		return fmt.Errorf("window management not supported on this platform")
	}
	if provider.Inputter == nil {
		return fmt.Errorf("input simulation not supported on this platform")
	}

	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")
	windowID, _ := cmd.Flags().GetInt("window-id")
	pid, _ := cmd.Flags().GetInt("pid")

	if appName == "" && window == "" && windowID == 0 && pid == 0 {
		return fmt.Errorf("specify --app, --window, --window-id, or --pid")
	}

	// Focus the target app
	if err := provider.WindowManager.FocusWindow(platform.FocusOptions{
		App:      appName,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
	}); err != nil {
		return fmt.Errorf("failed to focus app: %w", err)
	}

	// Brief pause for focus to settle
	time.Sleep(100 * time.Millisecond)

	// Select all (Cmd+A)
	if err := provider.Inputter.KeyCombo(strings.Split("cmd+a", "+")); err != nil {
		return fmt.Errorf("failed to select all: %w", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Copy (Cmd+C)
	if err := provider.Inputter.KeyCombo(strings.Split("cmd+c", "+")); err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Read clipboard
	text, err := provider.ClipboardManager.GetText()
	if err != nil {
		return err
	}

	return output.Print(ClipboardGrabResult{
		OK:     true,
		Action: "clipboard-grab",
		App:    appName,
		Text:   text,
	})
}
