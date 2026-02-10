package cmd

import (
	"fmt"
	"strings"

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

var findCmd = &cobra.Command{
	Use:   "find",
	Short: "Search for elements across all windows",
	Long:  "Search for UI elements by text across all windows (or all windows of a specific app). Useful when a dialog, notification, or new window appeared and you don't know which app owns it.",
	RunE:  runFind,
}

func init() {
	rootCmd.AddCommand(findCmd)
	findCmd.Flags().String("text", "", "Text to search for (case-insensitive substring match on title/value/description)")
	findCmd.Flags().String("roles", "", "Filter by role (e.g. \"btn\", \"btn,lnk,input\")")
	findCmd.Flags().String("app", "", "Limit search to windows of this app")
	findCmd.Flags().Int("limit", 10, "Max total matching elements to return")
	findCmd.Flags().Bool("exact", false, "Require exact match instead of substring")
}

// findWindowMatch groups matching elements with their source window info.
type findWindowMatch struct {
	App      string            `yaml:"app"               json:"app"`
	Window   string            `yaml:"window,omitempty"   json:"window,omitempty"`
	PID      int               `yaml:"pid"               json:"pid"`
	WindowID int               `yaml:"window_id"          json:"window_id"`
	Elements []findElementInfo `yaml:"elements"           json:"elements"`
}

// findElementInfo is a compact element representation for find results.
type findElementInfo struct {
	ID          int    `yaml:"i"               json:"i"`
	Role        string `yaml:"r"               json:"r"`
	Title       string `yaml:"t,omitempty"     json:"t,omitempty"`
	Value       string `yaml:"v,omitempty"     json:"v,omitempty"`
	Description string `yaml:"d,omitempty"     json:"d,omitempty"`
	Bounds      [4]int `yaml:"b"               json:"b"`
}

// findResult is the top-level output of the find command.
type findResult struct {
	OK      bool               `yaml:"ok"               json:"ok"`
	Action  string             `yaml:"action"           json:"action"`
	Text    string             `yaml:"text"             json:"text"`
	Matches []findWindowMatch  `yaml:"matches"          json:"matches"`
	Total   int                `yaml:"total"            json:"total"`
}

func runFind(cmd *cobra.Command, args []string) error {
	text, _ := cmd.Flags().GetString("text")
	rolesStr, _ := cmd.Flags().GetString("roles")
	appFilter, _ := cmd.Flags().GetString("app")
	limit, _ := cmd.Flags().GetInt("limit")
	exact, _ := cmd.Flags().GetBool("exact")

	if text == "" {
		return fmt.Errorf("--text is required")
	}

	provider, err := platform.NewProvider()
	if err != nil {
		return err
	}
	if provider.Reader == nil {
		return fmt.Errorf("reader not available on this platform")
	}

	// Build role set for filtering
	roleSet := make(map[string]bool)
	if rolesStr != "" {
		var roleList []string
		for _, r := range strings.Split(rolesStr, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				roleList = append(roleList, r)
			}
		}
		for _, r := range model.ExpandRoles(roleList) {
			roleSet[r] = true
		}
	}

	// List all windows, optionally filtered by app
	listOpts := platform.ListOptions{App: appFilter}
	windows, err := provider.Reader.ListWindows(listOpts)
	if err != nil {
		return fmt.Errorf("failed to list windows: %w", err)
	}

	// Sort: focused windows first for faster results in the common case
	sortWindowsFocusedFirst(windows)

	textLower := strings.ToLower(text)
	var matches []findWindowMatch
	totalFound := 0

	for _, win := range windows {
		if totalFound >= limit {
			break
		}

		elements, err := provider.Reader.ReadElements(platform.ReadOptions{
			PID:      win.PID,
			WindowID: win.ID,
		})
		if err != nil {
			continue // skip windows that fail to read
		}

		found := filterVisibleElements(collectLeafMatches(elements, textLower, roleSet, exact))
		if len(found) == 0 {
			continue
		}

		remaining := limit - totalFound
		if len(found) > remaining {
			found = found[:remaining]
		}

		var elInfos []findElementInfo
		for _, el := range found {
			elInfos = append(elInfos, findElementInfo{
				ID:          el.ID,
				Role:        el.Role,
				Title:       el.Title,
				Value:       el.Value,
				Description: el.Description,
				Bounds:      el.Bounds,
			})
		}

		matches = append(matches, findWindowMatch{
			App:      win.App,
			Window:   win.Title,
			PID:      win.PID,
			WindowID: win.ID,
			Elements: elInfos,
		})
		totalFound += len(elInfos)
	}

	result := findResult{
		OK:      true,
		Action:  "find",
		Text:    text,
		Matches: matches,
		Total:   totalFound,
	}

	if matches == nil {
		result.Matches = []findWindowMatch{}
	}

	return output.Print(result)
}

// sortWindowsFocusedFirst moves focused windows to the front of the slice.
func sortWindowsFocusedFirst(windows []model.Window) {
	i := 0
	for j := range windows {
		if windows[j].Focused {
			windows[i], windows[j] = windows[j], windows[i]
			i++
		}
	}
}
