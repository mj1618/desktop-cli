package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/output"
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

// directChildrenOnly returns a copy of elements with their children removed,
// effectively showing only the direct children of a parent element.
func directChildrenOnly(elements []model.Element) []model.Element {
	result := make([]model.Element, len(elements))
	for i, el := range elements {
		result[i] = el
		result[i].Children = nil
	}
	return result
}

// collectLeafMatches collects elements that directly match the text (case-insensitive
// substring on title/value/description), optionally filtered by role.
// It recurses into children but only returns the deepest (most specific) matches.
// If exact is true, only exact (case-insensitive) matches on title/value/description are used.
func collectLeafMatches(elements []model.Element, textLower string, roles map[string]bool, exact bool) []*model.Element {
	var results []*model.Element
	for i := range elements {
		el := &elements[i]

		// Check children first
		childMatches := collectLeafMatches(el.Children, textLower, roles, exact)

		selfMatch := textMatchesElement(*el, textLower, exact) && (len(roles) == 0 || roles[el.Role])

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

func textMatchesElement(el model.Element, textLower string, exact bool) bool {
	if exact {
		return exactFieldMatch(el.Title, textLower) ||
			exactFieldMatch(el.Value, textLower) ||
			exactFieldMatch(el.Description, textLower)
	}
	return strings.Contains(strings.ToLower(el.Title), textLower) ||
		strings.Contains(strings.ToLower(el.Value), textLower) ||
		strings.Contains(strings.ToLower(el.Description), textLower)
}

// exactFieldMatch returns true if field matches text case-insensitively,
// either directly or after stripping a trailing parenthetical suffix like " (⌘Enter)".
func exactFieldMatch(field, textLower string) bool {
	if strings.EqualFold(field, textLower) {
		return true
	}
	if idx := strings.LastIndex(field, "("); idx > 0 && strings.HasSuffix(strings.TrimRight(field, "\u202c"), ")") {
		stripped := strings.TrimRight(field[:idx], " \u202a")
		return strings.EqualFold(stripped, textLower)
	}
	return false
}

// resolveElementByText reads the element tree and finds a single element matching
// the given text (and optional role filter). Returns an error if zero or multiple
// elements match. The error for multiple matches includes candidate details so the
// agent can refine.
//
// If exact is true, only exact (case-insensitive) matches are considered.
// If scopeID > 0, only descendants of that element are searched.
// When multiple matches exist, elements closer to the focused element in the tree
// are preferred automatically.
func resolveElementByText(provider *platform.Provider, appName, window string, windowID, pid int, text string, roles string, exact bool, scopeID int) (*model.Element, []model.Element, error) {
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

	// Determine the search scope
	searchScope := elements
	if scopeID > 0 {
		scopeEl := findElementByID(elements, scopeID)
		if scopeEl == nil {
			return nil, elements, fmt.Errorf("scope element with id %d not found", scopeID)
		}
		searchScope = scopeEl.Children
	}

	textLower := strings.ToLower(text)

	roleSet := make(map[string]bool)
	if roles != "" {
		var roleList []string
		for _, r := range strings.Split(roles, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				roleList = append(roleList, r)
			}
		}
		for _, r := range model.ExpandRoles(roleList) {
			roleSet[r] = true
		}
	}

	matches := collectLeafMatches(searchScope, textLower, roleSet, exact)

	if len(matches) == 0 {
		return nil, elements, fmt.Errorf("no element found matching text %q", text)
	}
	if len(matches) == 1 {
		return matches[0], elements, nil
	}

	// Multiple matches — try to narrow down using focus proximity.
	// Elements sharing a deeper common ancestor with the focused element
	// (e.g. in the same dialog/modal) are preferred.
	matches = narrowByFocusProximity(elements, matches)
	if len(matches) == 1 {
		return matches[0], elements, nil
	}

	// Smart role prioritization: when --roles is not specified and matches
	// include both interactive (btn, lnk, input, etc.) and static (txt)
	// elements, prefer the interactive ones. This handles cases like
	// Calculator where "3" matches both a txt display and a btn.
	if roles == "" {
		matches = preferInteractiveElements(matches)
		if len(matches) == 1 {
			return matches[0], elements, nil
		}
	}

	// Still multiple matches — build a helpful error with enough context
	// (bounds, path) so the agent can distinguish elements without a follow-up read.
	var b strings.Builder
	fmt.Fprintf(&b, "multiple elements match text %q", text)
	if roles != "" {
		fmt.Fprintf(&b, " with roles %q", roles)
	}
	fmt.Fprintf(&b, " — use --id, --exact, or --scope-id to narrow:\n")
	for _, m := range matches {
		fmt.Fprintf(&b, "  id=%d %s (%d,%d,%d,%d)", m.ID, m.Role,
			m.Bounds[0], m.Bounds[1], m.Bounds[2], m.Bounds[3])
		if m.Title != "" {
			fmt.Fprintf(&b, " title=%q", m.Title)
		}
		if m.Description != "" {
			fmt.Fprintf(&b, " desc=%q", m.Description)
		}
		if path := findRolePathToID(elements, m.ID); path != "" {
			fmt.Fprintf(&b, " path=%q", path)
		}
		fmt.Fprintln(&b)
	}
	return nil, elements, fmt.Errorf("%s", b.String())
}

// narrowByFocusProximity filters matches to those sharing the deepest common
// ancestor with the currently focused element. This prefers elements in the
// same dialog/modal as the focused element over background elements.
// Returns the original matches if no focused element exists or if focus
// proximity doesn't help narrow the results.
func narrowByFocusProximity(elements []model.Element, matches []*model.Element) []*model.Element {
	focused := findFocusedElement(elements)
	if focused == nil {
		return matches
	}
	focusPath := findPathToID(elements, focused.ID)
	if len(focusPath) == 0 {
		return matches
	}

	bestScore := 0
	scores := make([]int, len(matches))
	for i, m := range matches {
		matchPath := findPathToID(elements, m.ID)
		scores[i] = commonPrefixLen(focusPath, matchPath)
		if scores[i] > bestScore {
			bestScore = scores[i]
		}
	}

	if bestScore == 0 {
		return matches
	}

	var narrowed []*model.Element
	for i, m := range matches {
		if scores[i] == bestScore {
			narrowed = append(narrowed, m)
		}
	}
	return narrowed
}

// staticRoles are display-only roles that should be deprioritized when
// interactive elements also match the same text.
var staticRoles = map[string]bool{
	"txt":   true,
	"img":   true,
	"group": true,
	"other": true,
}

// preferInteractiveElements filters matches to interactive (non-static)
// elements when the match set contains a mix of interactive and static roles.
// If ALL matches are static or ALL are interactive, returns the original set.
func preferInteractiveElements(matches []*model.Element) []*model.Element {
	var interactive []*model.Element
	for _, m := range matches {
		if !staticRoles[m.Role] {
			interactive = append(interactive, m)
		}
	}
	// Only filter if there's a mix — some interactive and some static
	if len(interactive) > 0 && len(interactive) < len(matches) {
		return interactive
	}
	return matches
}

// findPathToID returns the path (list of element IDs) from the root to the
// element with the given ID. Returns nil if not found.
func findPathToID(elements []model.Element, targetID int) []int {
	for i := range elements {
		if elements[i].ID == targetID {
			return []int{elements[i].ID}
		}
		if childPath := findPathToID(elements[i].Children, targetID); childPath != nil {
			return append([]int{elements[i].ID}, childPath...)
		}
	}
	return nil
}

// findRolePathToID returns the role-based path from root to the element with
// the given ID, e.g. "window > group > scroll > list > row > cell > txt".
// Returns "" if the element is not found.
func findRolePathToID(elements []model.Element, targetID int) string {
	parts := findRolePathParts(elements, targetID)
	if parts == nil {
		return ""
	}
	return strings.Join(parts, " > ")
}

func findRolePathParts(elements []model.Element, targetID int) []string {
	for i := range elements {
		if elements[i].ID == targetID {
			return []string{elements[i].Role}
		}
		if childPath := findRolePathParts(elements[i].Children, targetID); childPath != nil {
			return append([]string{elements[i].Role}, childPath...)
		}
	}
	return nil
}

// commonPrefixLen returns the length of the common prefix between two int slices.
func commonPrefixLen(a, b []int) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// collectAllElements recursively collects all elements in the tree into a flat slice.
func collectAllElements(elements []model.Element, result *[]*model.Element) {
	for i := range elements {
		*result = append(*result, &elements[i])
		collectAllElements(elements[i].Children, result)
	}
}

// nearMaxRadius is the maximum pixel distance (center-to-center) for the
// --near flag to consider an interactive element as genuinely "near" the
// text label. Beyond this radius, we fall back to offset-based clicking.
const nearMaxRadius = 200

// findNearestInteractiveElement finds the closest interactive element to the
// given anchor element, constrained by direction and radius.
//
// direction can be: "" (any direction, but prefer left), "left", "right",
// "above", or "below". When direction is "", the function first tries to find
// an element to the left (for checkbox patterns), then falls back to any
// direction within nearMaxRadius.
//
// Returns nil if no interactive element is found within the constraints.
func findNearestInteractiveElement(elements []model.Element, anchor *model.Element, direction string) *model.Element {
	anchorCX := anchor.Bounds[0] + anchor.Bounds[2]/2
	anchorCY := anchor.Bounds[1] + anchor.Bounds[3]/2

	var allElems []*model.Element
	collectAllElements(elements, &allElems)

	maxDistSq := int64(nearMaxRadius) * int64(nearMaxRadius)

	matchesDirection := func(cx, cy int) bool {
		switch direction {
		case "left":
			return cx < anchorCX
		case "right":
			return cx > anchorCX
		case "above":
			return cy < anchorCY
		case "below":
			return cy > anchorCY
		default:
			return true
		}
	}

	findBest := func(dirFilter func(cx, cy int) bool) *model.Element {
		var best *model.Element
		bestDist := int64(1<<62 - 1)
		for _, el := range allElems {
			if el.ID == anchor.ID {
				continue
			}
			if staticRoles[el.Role] {
				continue
			}
			// Skip elements with zero-size bounds (off-screen/virtualized)
			if el.Bounds[2] == 0 || el.Bounds[3] == 0 {
				continue
			}
			cx := el.Bounds[0] + el.Bounds[2]/2
			cy := el.Bounds[1] + el.Bounds[3]/2
			dx := int64(cx - anchorCX)
			dy := int64(cy - anchorCY)
			dist := dx*dx + dy*dy
			if dist > maxDistSq {
				continue
			}
			if !dirFilter(cx, cy) {
				continue
			}
			if dist < bestDist {
				bestDist = dist
				best = el
			}
		}
		return best
	}

	if direction != "" {
		return findBest(matchesDirection)
	}

	// Default: try left first (checkbox pattern), then any direction
	if best := findBest(func(cx, cy int) bool { return cx < anchorCX }); best != nil {
		return best
	}
	return findBest(func(cx, cy int) bool { return true })
}

// nearFallbackOffset returns click coordinates when no interactive element
// was found in the accessibility tree near the anchor. It uses the
// anchor element's bounding box and the specified direction to compute a
// reasonable click point (e.g. to the left of the text for checkboxes).
func nearFallbackOffset(anchor *model.Element, direction string) (x, y int) {
	bx, by, bw, bh := anchor.Bounds[0], anchor.Bounds[1], anchor.Bounds[2], anchor.Bounds[3]
	cy := by + bh/2
	cx := bx + bw/2

	switch direction {
	case "right":
		return bx + bw + 20, cy
	case "above":
		return cx, by - 20
	case "below":
		return cx, by + bh + 20
	default: // "", "left"
		// Click 20px to the left of the element's left edge, vertically centered
		return bx - 20, cy
	}
}

// addTextTargetingFlags adds --text, --roles, --exact, and --scope-id flags to a
// command for text-based element targeting. Use textFlagName to customize the flag
// name (e.g. "target" for the type command where --text is already used).
func addTextTargetingFlags(cmd *cobra.Command, textFlagName string, textHelp string) {
	cmd.Flags().String(textFlagName, "", textHelp)
	cmd.Flags().String("roles", "", "Filter by role when using text targeting (e.g. \"btn\", \"btn,lnk\")")
	cmd.Flags().Bool("exact", false, "Require exact match on title/value/description (default: substring)")
	cmd.Flags().Int("scope-id", 0, "Limit text search to descendants of this element ID")
}

// getTextTargetingFlags reads the text-targeting flags from a command.
func getTextTargetingFlags(cmd *cobra.Command, textFlagName string) (text string, roles string, exact bool, scopeID int) {
	text, _ = cmd.Flags().GetString(textFlagName)
	roles, _ = cmd.Flags().GetString("roles")
	exact, _ = cmd.Flags().GetBool("exact")
	scopeID, _ = cmd.Flags().GetInt("scope-id")
	return
}

// ElementInfo is a compact representation of a single UI element, used in
// type/action/key responses to report the target or focused element.
type ElementInfo struct {
	ID          int    `yaml:"i"               json:"i"`
	Role        string `yaml:"r"               json:"r"`
	Title       string `yaml:"t,omitempty"     json:"t,omitempty"`
	Value       string `yaml:"v,omitempty"     json:"v,omitempty"`
	Description string `yaml:"d,omitempty"     json:"d,omitempty"`
	Bounds      [4]int `yaml:"b"               json:"b"`
	Primary     bool   `yaml:"primary,omitempty" json:"primary,omitempty"`
}

// elementInfoFromElement converts an Element to a compact ElementInfo.
func elementInfoFromElement(el *model.Element) *ElementInfo {
	return &ElementInfo{
		ID:          el.ID,
		Role:        el.Role,
		Title:       el.Title,
		Value:       el.Value,
		Description: el.Description,
		Bounds:      el.Bounds,
	}
}

// findFocusedElement searches the element tree for the element with f:true.
// Returns nil if no focused element is found.
func findFocusedElement(elements []model.Element) *ElementInfo {
	for i := range elements {
		if elements[i].Focused {
			return elementInfoFromElement(&elements[i])
		}
		if found := findFocusedElement(elements[i].Children); found != nil {
			return found
		}
	}
	return nil
}

// readFocusedElement reads the element tree and returns the currently focused element.
// If appName is empty, it tries to detect the frontmost app automatically.
// Returns nil if no focused element is found (best-effort, never errors).
func readFocusedElement(provider *platform.Provider, appName, window string, windowID, pid int) *ElementInfo {
	if provider.Reader == nil {
		return nil
	}

	// Determine app scope: use provided values, or detect frontmost app
	if appName == "" && window == "" && windowID == 0 && pid == 0 && provider.WindowManager != nil {
		if name, p, err := provider.WindowManager.GetFrontmostApp(); err == nil {
			appName = name
			pid = p
		}
	}
	if appName == "" && pid == 0 && windowID == 0 {
		return nil
	}

	elements, err := provider.Reader.ReadElements(platform.ReadOptions{
		App:      appName,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
	})
	if err != nil {
		return nil
	}
	return findFocusedElement(elements)
}

// readElementByID re-reads the element tree and returns info for a specific element.
// Returns nil if the element is not found (best-effort, never errors).
func readElementByID(provider *platform.Provider, appName, window string, windowID, pid, id int) *ElementInfo {
	if provider.Reader == nil {
		return nil
	}
	elements, err := provider.Reader.ReadElements(platform.ReadOptions{
		App:      appName,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
	})
	if err != nil {
		return nil
	}
	el := findElementByID(elements, id)
	if el == nil {
		return nil
	}
	return elementInfoFromElement(el)
}

// isDisplayElement returns true if the element is a read-only display text
// (role "txt" with a value, no "press" action) — the same criteria used by
// the agent output format's "display" flag.
func isDisplayElement(el model.Element) bool {
	if el.Role != "txt" || el.Value == "" {
		return false
	}
	for _, a := range el.Actions {
		if a == "press" {
			return false
		}
	}
	return true
}

// collectDisplayElements recursively finds all display elements in the tree.
func collectDisplayElements(elements []model.Element) []*model.Element {
	var results []*model.Element
	for i := range elements {
		if isDisplayElement(elements[i]) {
			results = append(results, &elements[i])
		}
		results = append(results, collectDisplayElements(elements[i].Children)...)
	}
	return results
}

// maxDisplayElements is the maximum number of display elements returned in
// click/action/type responses. Apps like Notes can have hundreds of text
// elements; capping prevents excessive output that wastes tokens.
const maxDisplayElements = 20

// readDisplayElements re-reads the element tree and returns ElementInfo for all
// display elements (read-only text with values). Returns nil if none found.
// When multiple display elements exist, the one with the largest bounding-box
// height is marked primary:true (e.g. the Calculator result vs. expression).
// Results are capped at maxDisplayElements to avoid excessive output.
func readDisplayElements(provider *platform.Provider, appName, window string, windowID, pid int) []ElementInfo {
	if provider.Reader == nil {
		return nil
	}
	elements, err := provider.Reader.ReadElements(platform.ReadOptions{
		App:      appName,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
	})
	if err != nil {
		return nil
	}
	displays := collectDisplayElements(elements)
	if len(displays) == 0 {
		return nil
	}
	if len(displays) > maxDisplayElements {
		displays = displays[:maxDisplayElements]
	}
	infos := make([]ElementInfo, len(displays))
	for i, el := range displays {
		infos[i] = *elementInfoFromElement(el)
	}
	// Mark the display element with the largest height as primary.
	// Height correlates with font size, so the tallest element is typically
	// the main result (e.g. Calculator result vs. expression).
	if len(infos) > 1 {
		maxH := -1
		maxIdx := 0
		for i, info := range infos {
			if info.Bounds[3] > maxH {
				maxH = info.Bounds[3]
				maxIdx = i
			}
		}
		infos[maxIdx].Primary = true
	}
	return infos
}

// requireScope checks that at least one scoping flag is set.
func requireScope(appName, window string, windowID, pid int) error {
	if appName == "" && window == "" && windowID == 0 && pid == 0 {
		return fmt.Errorf("--app, --window, --window-id, or --pid is required to scope the element lookup")
	}
	return nil
}

// addPostReadFlags adds --post-read and --post-read-delay flags to a command.
func addPostReadFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("post-read", false, "Include compact UI state (agent format) in the response after the action")
	cmd.Flags().Int("post-read-delay", 0, "Delay in ms before reading UI state (use for actions that trigger animations/transitions)")
}

// getPostReadFlags reads --post-read and --post-read-delay from a command.
func getPostReadFlags(cmd *cobra.Command) (postRead bool, postReadDelay int) {
	postRead, _ = cmd.Flags().GetBool("post-read")
	postReadDelay, _ = cmd.Flags().GetInt("post-read-delay")
	return
}

// readPostActionState reads the UI tree after an action and returns a compact
// agent-format string. Best-effort: returns "" if the read fails.
func readPostActionState(provider *platform.Provider, appName, window string, windowID, pid int, delayMs int) string {
	if provider.Reader == nil {
		return ""
	}
	if delayMs > 0 {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}
	elements, err := provider.Reader.ReadElements(platform.ReadOptions{
		App:      appName,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
	})
	if err != nil {
		return ""
	}

	// Resolve window title from the element tree
	windowTitle := window
	if windowTitle == "" {
		for _, el := range elements {
			if el.Role == "window" && el.Title != "" {
				windowTitle = el.Title
				break
			}
		}
	}

	return output.FormatAgentString(appName, pid, windowTitle, elements)
}
