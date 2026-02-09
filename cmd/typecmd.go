package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// TypeResult is the YAML output of a successful type command.
type TypeResult struct {
	OK     bool   `yaml:"ok"             json:"ok"`
	Action string `yaml:"action"         json:"action"`
	Text   string `yaml:"text,omitempty" json:"text,omitempty"`
	Key    string `yaml:"key,omitempty"  json:"key,omitempty"`
}

var typeCmd = &cobra.Command{
	Use:   "type [text]",
	Short: "Type text or press key combinations",
	Long:  "Type text into the focused element or press key combinations. Text can be passed as a positional argument or via --text.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runType,
}

func init() {
	rootCmd.AddCommand(typeCmd)
	typeCmd.Flags().String("text", "", "Text to type (alternative to positional arg)")
	typeCmd.Flags().String("key", "", "Key combination (e.g. \"cmd+c\", \"ctrl+shift+t\", \"enter\", \"tab\")")
	typeCmd.Flags().Int("delay", 0, "Delay between keystrokes in ms")
	typeCmd.Flags().Int("id", 0, "Focus element by ID first, then type")
	typeCmd.Flags().String("app", "", "Scope to application (used with --id or --target)")
	typeCmd.Flags().String("window", "", "Scope to window (used with --id or --target)")
	addTextTargetingFlags(typeCmd, "target", "Find element by text and focus it before typing (case-insensitive match on title/value/description)")
}

func runType(cmd *cobra.Command, args []string) error {
	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.Inputter == nil {
		return fmt.Errorf("input simulation not available on this platform")
	}

	text, _ := cmd.Flags().GetString("text")
	key, _ := cmd.Flags().GetString("key")
	delayMs, _ := cmd.Flags().GetInt("delay")
	id, _ := cmd.Flags().GetInt("id")
	appName, _ := cmd.Flags().GetString("app")
	window, _ := cmd.Flags().GetString("window")

	// Positional arg overrides --text flag
	if len(args) > 0 {
		text = args[0]
	}

	if text == "" && key == "" {
		return fmt.Errorf("specify --text, --key, or a positional text argument")
	}

	target, roles := getTextTargetingFlags(cmd, "target")
	hasTarget := target != ""

	// If --target or --id specified, click the element first to focus it
	if hasTarget {
		if appName == "" && window == "" {
			return fmt.Errorf("--target requires --app or --window to scope the element lookup")
		}
		elem, _, err := resolveElementByText(provider, appName, window, 0, 0, target, roles)
		if err != nil {
			return err
		}
		cx := elem.Bounds[0] + elem.Bounds[2]/2
		cy := elem.Bounds[1] + elem.Bounds[3]/2
		if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
			return fmt.Errorf("failed to focus element: %w", err)
		}
		time.Sleep(50 * time.Millisecond)
	} else if id > 0 {
		if appName == "" && window == "" {
			return fmt.Errorf("--id requires --app or --window to scope the element lookup")
		}
		if provider.Reader == nil {
			return fmt.Errorf("reader not available on this platform")
		}
		elements, err := provider.Reader.ReadElements(platform.ReadOptions{
			App:    appName,
			Window: window,
		})
		if err != nil {
			return fmt.Errorf("failed to read elements: %w", err)
		}
		elem := findElementByID(elements, id)
		if elem == nil {
			return fmt.Errorf("element with id %d not found", id)
		}
		cx := elem.Bounds[0] + elem.Bounds[2]/2
		cy := elem.Bounds[1] + elem.Bounds[3]/2
		if err := provider.Inputter.Click(cx, cy, platform.MouseLeft, 1); err != nil {
			return fmt.Errorf("failed to focus element: %w", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	if key != "" {
		keys := strings.Split(key, "+")
		if err := provider.Inputter.KeyCombo(keys); err != nil {
			return err
		}
		return output.Print(TypeResult{
			OK:     true,
			Action: "key",
			Key:    key,
		})
	}

	if err := provider.Inputter.TypeText(text, delayMs); err != nil {
		return err
	}
	return output.Print(TypeResult{
		OK:     true,
		Action: "type",
		Text:   text,
	})
}
