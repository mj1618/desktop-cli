package cmd

import (
	"fmt"
	"strings"

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// findElementByID searches the element tree recursively for an element with the given ID.
func findElementByID(elements []model.Element, id int) *model.Element {
	for i := range elements {
		if elements[i].ID == id {
			return &elements[i]
		}
		if found := findElementByID(elements[i].Children, id); found != nil {
			return found
		}
	}
	return nil
}

// collectLeafMatches collects elements that directly match the text (case-insensitive
// substring on title/value/description), optionally filtered by role.
// It recurses into children but only returns the deepest (most specific) matches.
func collectLeafMatches(elements []model.Element, textLower string, roles map[string]bool) []*model.Element {
	var results []*model.Element
	for i := range elements {
		el := &elements[i]

		// Check children first
		childMatches := collectLeafMatches(el.Children, textLower, roles)

		selfMatch := textMatchesElement(*el, textLower) && (len(roles) == 0 || roles[el.Role])

		if selfMatch && len(childMatches) == 0 {
			// Leaf match: this element matches and none of its children do
			results = append(results, el)
		} else {
			// Propagate child matches up
			results = append(results, childMatches...)
		}
	}
	return results
}

func textMatchesElement(el model.Element, textLower string) bool {
	return strings.Contains(strings.ToLower(el.Title), textLower) ||
		strings.Contains(strings.ToLower(el.Value), textLower) ||
		strings.Contains(strings.ToLower(el.Description), textLower)
}

// resolveElementByText reads the element tree and finds a single element matching
// the given text (and optional role filter). Returns an error if zero or multiple
// elements match. The error for multiple matches includes candidate details so the
// agent can refine.
func resolveElementByText(provider *platform.Provider, appName, window string, windowID, pid int, text string, roles string) (*model.Element, []model.Element, error) {
	if provider.Reader == nil {
		return nil, nil, fmt.Errorf("reader not available on this platform")
	}

	opts := platform.ReadOptions{
		App:      appName,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
	}
	elements, err := provider.Reader.ReadElements(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read elements: %w", err)
	}

	textLower := strings.ToLower(text)

	roleSet := make(map[string]bool)
	if roles != "" {
		for _, r := range strings.Split(roles, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				roleSet[r] = true
			}
		}
	}

	matches := collectLeafMatches(elements, textLower, roleSet)

	if len(matches) == 0 {
		return nil, elements, fmt.Errorf("no element found matching text %q", text)
	}
	if len(matches) == 1 {
		return matches[0], elements, nil
	}

	// Multiple matches — build a helpful error
	var b strings.Builder
	fmt.Fprintf(&b, "multiple elements match text %q", text)
	if roles != "" {
		fmt.Fprintf(&b, " with roles %q", roles)
	}
	fmt.Fprintf(&b, " — use --id to target a specific one:\n")
	for _, m := range matches {
		fmt.Fprintf(&b, "  id=%d role=%s", m.ID, m.Role)
		if m.Title != "" {
			fmt.Fprintf(&b, " title=%q", m.Title)
		}
		if m.Description != "" {
			fmt.Fprintf(&b, " desc=%q", m.Description)
		}
		fmt.Fprintln(&b)
	}
	return nil, elements, fmt.Errorf("%s", b.String())
}

// addTextTargetingFlags adds --text and --roles flags to a command for text-based
// element targeting. Use textFlagName to customize the flag name (e.g. "target" for
// the type command where --text is already used).
func addTextTargetingFlags(cmd *cobra.Command, textFlagName string, textHelp string) {
	cmd.Flags().String(textFlagName, "", textHelp)
	cmd.Flags().String("roles", "", "Filter by role when using text targeting (e.g. \"btn\", \"btn,lnk\")")
}

// getTextTargetingFlags reads the text-targeting flags from a command.
func getTextTargetingFlags(cmd *cobra.Command, textFlagName string) (text string, roles string) {
	text, _ = cmd.Flags().GetString(textFlagName)
	roles, _ = cmd.Flags().GetString("roles")
	return
}

// requireScope checks that at least one scoping flag is set.
func requireScope(appName, window string, windowID, pid int) error {
	if appName == "" && window == "" && windowID == 0 && pid == 0 {
		return fmt.Errorf("--app, --window, --window-id, or --pid is required to scope the element lookup")
	}
	return nil
}
