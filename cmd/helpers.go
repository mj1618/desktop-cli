package cmd

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// noAutoScope disables automatic overlay/dialog scoping when set to true.
// Set by the --no-auto-scope flag on commands that use text-based targeting.
var noAutoScope bool

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

// filterVisibleElements removes elements with zero width or zero height
// (off-screen/virtualized elements that aren't actually visible or clickable).
// This prevents false ambiguity in text-based targeting when zero-dimension
// elements match the same text as visible ones.
func filterVisibleElements(matches []*model.Element) []*model.Element {
	var visible []*model.Element
	for _, m := range matches {
		if m.Bounds[2] > 0 && m.Bounds[3] > 0 {
			visible = append(visible, m)
		}
	}
	return visible
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
//
// Auto-scope: When scopeID is 0 and a dialog/sheet/popover is detected, the search
// is automatically scoped to the overlay's subtree. This eliminates false matches
// from background elements behind modals. Use --no-auto-scope to disable.
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

	// Auto-scope: when no manual scope is set, detect overlay and scope to it
	autoScoped := false
	var overlayScope []model.Element
	if scopeID == 0 && !noAutoScope {
		if overlay := model.DetectFrontmostOverlay(elements); overlay != nil {
			overlayScope = overlay.Children
			autoScoped = true
		}
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

	// If auto-scoped, try searching in the overlay first
	if autoScoped {
		scopedMatches := filterVisibleElements(collectLeafMatches(overlayScope, textLower, roleSet, exact))
		if len(scopedMatches) > 0 {
			// Use the scoped matches — resolve from within the overlay
			result, resultErr := narrowMatches(elements, scopedMatches, text, roles)
			if resultErr == nil {
				return result, elements, nil
			}
			// If scoped search had multiple matches, still use them (better than full tree)
			return nil, elements, resultErr
		}
		// No matches in overlay — fall through to full tree search
	}

	matches := filterVisibleElements(collectLeafMatches(searchScope, textLower, roleSet, exact))

	if len(matches) == 0 {
		if autoScoped {
			return nil, elements, fmt.Errorf("no element found matching text %q (auto-scoped to dialog — try --no-auto-scope to search entire window)", text)
		}
		return nil, elements, fmt.Errorf("no element found matching text %q", text)
	}

	result, resultErr := narrowMatches(elements, matches, text, roles)
	if resultErr == nil {
		return result, elements, nil
	}
	return nil, elements, resultErr
}

// narrowMatches applies focus proximity and interactive element preference to
// narrow a set of matches down to a single result. Returns the element if
// uniquely resolved, or an error with candidate details.
func narrowMatches(elements []model.Element, matches []*model.Element, text, roles string) (*model.Element, error) {
	if len(matches) == 1 {
		return matches[0], nil
	}

	// Multiple matches — try to narrow down using focus proximity.
	matches = narrowByFocusProximity(elements, matches)
	if len(matches) == 1 {
		return matches[0], nil
	}

	// Smart role prioritization
	if roles == "" {
		matches = preferInteractiveElements(matches)
		if len(matches) == 1 {
			return matches[0], nil
		}
	}

	// Still multiple matches — build a helpful error
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
	return nil, fmt.Errorf("%s", b.String())
}

// resolveElementByTextFromTree finds a single element matching the given text
// from a pre-read element tree, avoiding a redundant tree read. Uses the same
// matching logic as resolveElementByText, including auto-scoping to overlays.
func resolveElementByTextFromTree(elements []model.Element, text string, roles string, exact bool, scopeID int) (*model.Element, error) {
	searchScope := elements
	if scopeID > 0 {
		scopeEl := findElementByID(elements, scopeID)
		if scopeEl == nil {
			return nil, fmt.Errorf("scope element with id %d not found", scopeID)
		}
		searchScope = scopeEl.Children
	}

	// Auto-scope to overlay when no manual scope is set
	autoScoped := false
	var overlayScope []model.Element
	if scopeID == 0 && !noAutoScope {
		if overlay := model.DetectFrontmostOverlay(elements); overlay != nil {
			overlayScope = overlay.Children
			autoScoped = true
		}
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

	// If auto-scoped, try overlay first
	if autoScoped {
		scopedMatches := filterVisibleElements(collectLeafMatches(overlayScope, textLower, roleSet, exact))
		if len(scopedMatches) > 0 {
			return narrowMatchesSimple(elements, scopedMatches, text, roles)
		}
		// No matches in overlay — fall through to full tree search
	}

	matches := filterVisibleElements(collectLeafMatches(searchScope, textLower, roleSet, exact))

	if len(matches) == 0 {
		return nil, fmt.Errorf("no element found matching text %q", text)
	}

	return narrowMatchesSimple(elements, matches, text, roles)
}

// narrowMatchesSimple is like narrowMatches but returns (*Element, error) without the elements slice.
func narrowMatchesSimple(elements []model.Element, matches []*model.Element, text, roles string) (*model.Element, error) {
	if len(matches) == 1 {
		return matches[0], nil
	}

	matches = narrowByFocusProximity(elements, matches)
	if len(matches) == 1 {
		return matches[0], nil
	}

	if roles == "" {
		matches = preferInteractiveElements(matches)
		if len(matches) == 1 {
			return matches[0], nil
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "multiple elements match text %q", text)
	if roles != "" {
		fmt.Fprintf(&b, " with roles %q", roles)
	}
	fmt.Fprintf(&b, " — use id: to target a specific element:\n")
	for _, m := range matches {
		fmt.Fprintf(&b, "  id=%d %s (%d,%d,%d,%d)", m.ID, m.Role,
			m.Bounds[0], m.Bounds[1], m.Bounds[2], m.Bounds[3])
		if m.Title != "" {
			fmt.Fprintf(&b, " title=%q", m.Title)
		}
		if m.Description != "" {
			fmt.Fprintf(&b, " desc=%q", m.Description)
		}
		fmt.Fprintln(&b)
	}
	return nil, fmt.Errorf("%s", b.String())
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

// resolveAllTextMatches reads the element tree and returns ALL elements matching
// the given text (and optional role filter), plus the full element tree.
// Unlike resolveElementByText, this never errors on multiple matches — it returns
// them all. Used by the --near path to evaluate all candidates.
func resolveAllTextMatches(provider *platform.Provider, appName, window string, windowID, pid int, text string, roles string, exact bool, scopeID int) ([]*model.Element, []model.Element, error) {
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

	searchScope := elements
	if scopeID > 0 {
		scopeEl := findElementByID(elements, scopeID)
		if scopeEl == nil {
			return nil, elements, fmt.Errorf("scope element with id %d not found", scopeID)
		}
		searchScope = scopeEl.Children
	}

	// Auto-scope: try overlay first
	autoScoped := false
	var overlayScope []model.Element
	if scopeID == 0 && !noAutoScope {
		if overlay := model.DetectFrontmostOverlay(elements); overlay != nil {
			overlayScope = overlay.Children
			autoScoped = true
		}
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

	// If auto-scoped, try overlay first
	if autoScoped {
		scopedMatches := filterVisibleElements(collectLeafMatches(overlayScope, textLower, roleSet, exact))
		if len(scopedMatches) > 0 {
			return scopedMatches, elements, nil
		}
	}

	matches := filterVisibleElements(collectLeafMatches(searchScope, textLower, roleSet, exact))
	if len(matches) == 0 {
		return nil, elements, fmt.Errorf("no element found matching text %q", text)
	}

	return matches, elements, nil
}

// pickBestNearMatch selects the best text match for --near when multiple
// candidates exist. It prefers matches in the main content area by choosing
// the rightmost match (macOS apps typically use left sidebar + right content).
// Among matches at similar X positions (within 50px), the one closest to the
// focused element wins.
func pickBestNearMatch(elements []model.Element, matches []*model.Element) *model.Element {
	if len(matches) == 1 {
		return matches[0]
	}

	// Sort by X position (descending) to prefer rightmost (content area)
	type scored struct {
		el *model.Element
		x  int
	}
	scored_list := make([]scored, len(matches))
	for i, m := range matches {
		scored_list[i] = scored{el: m, x: m.Bounds[0]}
	}
	sort.Slice(scored_list, func(i, j int) bool {
		return scored_list[i].x > scored_list[j].x
	})

	// Group matches within 50px of the rightmost X position
	rightmostX := scored_list[0].x
	var rightGroup []*model.Element
	for _, s := range scored_list {
		if rightmostX-s.x <= 50 {
			rightGroup = append(rightGroup, s.el)
		}
	}

	if len(rightGroup) == 1 {
		return rightGroup[0]
	}

	// Multiple matches in the rightmost group — use focus proximity
	narrowed := narrowByFocusProximity(elements, rightGroup)
	if len(narrowed) >= 1 {
		return narrowed[0]
	}
	return rightGroup[0]
}

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

// addTextTargetingFlags adds --text, --roles, --exact, --scope-id, and --no-auto-scope
// flags to a command for text-based element targeting. Use textFlagName to customize the
// flag name (e.g. "target" for the type command where --text is already used).
func addTextTargetingFlags(cmd *cobra.Command, textFlagName string, textHelp string) {
	cmd.Flags().String(textFlagName, "", textHelp)
	cmd.Flags().String("roles", "", "Filter by role when using text targeting (e.g. \"btn\", \"btn,lnk\")")
	cmd.Flags().Bool("exact", false, "Require exact match on title/value/description (default: substring)")
	cmd.Flags().Int("scope-id", 0, "Limit text search to descendants of this element ID")
	cmd.Flags().Bool("no-auto-scope", false, "Disable automatic scoping to frontmost dialog/sheet/popover")
}

// getTextTargetingFlags reads the text-targeting flags from a command.
// Also sets the package-level noAutoScope variable.
func getTextTargetingFlags(cmd *cobra.Command, textFlagName string) (text string, roles string, exact bool, scopeID int) {
	text, _ = cmd.Flags().GetString(textFlagName)
	noAutoScope, _ = cmd.Flags().GetBool("no-auto-scope")
	roles, _ = cmd.Flags().GetString("roles")
	exact, _ = cmd.Flags().GetBool("exact")
	scopeID, _ = cmd.Flags().GetInt("scope-id")
	return
}

// addRefFlag adds the --ref flag to a command for stable ref-based targeting.
func addRefFlag(cmd *cobra.Command) {
	cmd.Flags().String("ref", "", "Target element by stable ref from a prior read (e.g. \"toolbar/search\", \"dialog/submit\")")
}

// resolveElementByRef reads the element tree, generates refs, and finds the
// element matching the given ref. Supports exact and partial suffix matching.
func resolveElementByRef(provider *platform.Provider, appName, window string, windowID, pid int, ref string) (*model.Element, []model.Element, error) {
	if provider.Reader == nil {
		return nil, nil, fmt.Errorf("reader not available on this platform")
	}

	elements, err := provider.Reader.ReadElements(platform.ReadOptions{
		App:      appName,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read elements: %w", err)
	}

	model.GenerateRefs(elements)

	el, err := model.FindElementByRef(elements, ref)
	if err != nil {
		return nil, elements, err
	}
	return el, elements, nil
}

// resolveElementByRefFromTree finds an element by ref from a pre-read tree
// (refs must already be generated).
func resolveElementByRefFromTree(elements []model.Element, ref string) (*model.Element, error) {
	return model.FindElementByRef(elements, ref)
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

// findFocusedElement searches the element tree for the deepest focused element.
// When multiple elements report Focused=true (common in apps like Reminders where
// parent cells and the actual input both report focus), the deepest (leaf-most)
// element is returned since it's the one actually receiving keyboard input.
// Among equally deep focused elements, input roles are preferred.
// Returns nil if no focused element is found.
func findFocusedElement(elements []model.Element) *ElementInfo {
	el := findDeepestFocused(elements)
	if el == nil {
		return nil
	}
	return elementInfoFromElement(el)
}

// findDeepestFocused returns the deepest focused element in the tree.
// If a focused element has focused descendants, the descendant is preferred.
// Among siblings at the same level, input roles are preferred.
func findDeepestFocused(elements []model.Element) *model.Element {
	var best *model.Element
	for i := range elements {
		// Always recurse into children first to find deeper focused elements
		if childFocused := findDeepestFocused(elements[i].Children); childFocused != nil {
			if best == nil || isMoreSpecificFocus(childFocused, best) {
				best = childFocused
			}
		} else if elements[i].Focused {
			// This element is focused and has no focused descendants
			if best == nil || isMoreSpecificFocus(&elements[i], best) {
				best = &elements[i]
			}
		}
	}
	return best
}

// editableRoles are roles that can receive text input.
var editableRoles = map[string]bool{
	"input": true,
	"other": true, // web inputs sometimes mapped as "other"
}

// isMoreSpecificFocus returns true if candidate is a better focused element
// than current. Prefers input roles over non-input roles.
func isMoreSpecificFocus(candidate, current *model.Element) bool {
	candidateEditable := editableRoles[candidate.Role]
	currentEditable := editableRoles[current.Role]
	if candidateEditable && !currentEditable {
		return true
	}
	return false
}

// readFocusedElement reads the element tree and returns the currently focused element.
// If appName is empty, it tries to detect the frontmost app automatically.
// Returns nil if no focused element is found (best-effort, never errors).
func readFocusedElement(provider *platform.Provider, appName, window string, windowID, pid int) *ElementInfo {
	if provider == nil || provider.Reader == nil {
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

// distanceToBounds returns the Euclidean distance from the center of element el
// to the center of targetBounds. Used to prioritize display elements near the
// action target.
func distanceToBounds(el *model.Element, targetBounds [4]int) float64 {
	elCX := float64(el.Bounds[0]) + float64(el.Bounds[2])/2
	elCY := float64(el.Bounds[1]) + float64(el.Bounds[3])/2
	tCX := float64(targetBounds[0]) + float64(targetBounds[2])/2
	tCY := float64(targetBounds[1]) + float64(targetBounds[3])/2
	dx := elCX - tCX
	dy := elCY - tCY
	return math.Sqrt(dx*dx + dy*dy)
}

// readDisplayElements re-reads the element tree and returns ElementInfo for all
// display elements (read-only text with values). Returns nil if none found.
// When multiple display elements exist, the one with the largest bounding-box
// height is marked primary:true (e.g. the Calculator result vs. expression).
// Results are capped at maxDisplayElements to avoid excessive output.
//
// If targetBounds is non-zero, display elements are sorted by proximity to the
// target before capping, so the most contextually relevant elements are kept.
func readDisplayElements(provider *platform.Provider, appName, window string, windowID, pid int, targetBounds [4]int) []ElementInfo {
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
	// When there are more display elements than the cap and we know the target
	// location, sort by proximity so the most relevant elements survive the cap.
	hasTarget := targetBounds[2] > 0 || targetBounds[3] > 0
	if len(displays) > maxDisplayElements && hasTarget {
		sort.Slice(displays, func(i, j int) bool {
			return distanceToBounds(displays[i], targetBounds) < distanceToBounds(displays[j], targetBounds)
		})
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

// addPostReadFlags adds --post-read, --post-read-delay, and --post-read-max-elements flags to a command.
func addPostReadFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("post-read", false, "Include compact UI state (agent format) in the response after the action")
	cmd.Flags().Int("post-read-delay", 0, "Delay in ms before reading UI state (use for actions that trigger animations/transitions)")
	cmd.Flags().Int("post-read-max-elements", 0, "Max elements in post-read output (default: 200 for web content, 0=unlimited for native apps)")
}

// postReadOptions holds the parameters for --post-read behavior.
type postReadOptions struct {
	PostRead    bool
	Delay       int // ms
	MaxElements int // 0 = auto (200 for web, unlimited for native)
}

// getPostReadFlags reads --post-read, --post-read-delay, and --post-read-max-elements from a command.
func getPostReadFlags(cmd *cobra.Command) (postRead bool, postReadDelay int) {
	postRead, _ = cmd.Flags().GetBool("post-read")
	postReadDelay, _ = cmd.Flags().GetInt("post-read-delay")
	return
}

// getPostReadOptions reads all post-read flags from a command.
func getPostReadOptions(cmd *cobra.Command) postReadOptions {
	postRead, _ := cmd.Flags().GetBool("post-read")
	delay, _ := cmd.Flags().GetInt("post-read-delay")
	maxElements, _ := cmd.Flags().GetInt("post-read-max-elements")
	return postReadOptions{
		PostRead:    postRead,
		Delay:       delay,
		MaxElements: maxElements,
	}
}

// defaultPostReadMaxElements is the max elements applied to post-read output
// for web content, matching the read command's smart default. This prevents
// 30KB+ output on complex web pages that would trigger file-based storage.
const defaultPostReadMaxElements = 200

// readPostActionState reads the UI tree after an action and returns a compact
// agent-format string. Best-effort: returns "" if the read fails.
// maxElements controls the element cap: 0 means auto (200 for web, unlimited for native).
func readPostActionState(provider *platform.Provider, appName, window string, windowID, pid int, delayMs int, maxElements int) string {
	_, formatted := readPostActionElements(provider, appName, window, windowID, pid, delayMs, maxElements)
	return formatted
}

// readPostActionElements reads the UI tree after an action, returning both the
// raw element tree and the formatted agent string. The raw elements can be
// reused by --verify to avoid a redundant tree read. Best-effort: returns nil
// elements and "" if the read fails.
// maxElements controls the element cap: 0 means auto (200 for web, unlimited for native).
func readPostActionElements(provider *platform.Provider, appName, window string, windowID, pid int, delayMs int, maxElements int) ([]model.Element, string) {
	if provider.Reader == nil {
		return nil, ""
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
		return nil, ""
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

	// Auto-apply max-elements for web content (same as read command smart default)
	effectiveMax := maxElements
	if effectiveMax == 0 && model.HasWebContent(elements) {
		effectiveMax = defaultPostReadMaxElements
	}

	return elements, output.FormatAgentStringWithMax(appName, pid, windowTitle, elements, effectiveMax)
}

// --- Verification infrastructure ---

// elementSnapshot captures the state of a UI element before an action,
// used by --verify to detect whether the action caused a change.
type elementSnapshot struct {
	ID         int
	Title      string
	Value      string
	Focused    bool
	Selected   bool
	Bounds     [4]int
	ChildCount int
	Exists     bool // true when the element was found pre-action
}

// snapshotElement captures the verifiable state of an element.
func snapshotElement(el *model.Element) elementSnapshot {
	return elementSnapshot{
		ID:         el.ID,
		Title:      el.Title,
		Value:      el.Value,
		Focused:    el.Focused,
		Selected:   el.Selected,
		Bounds:     el.Bounds,
		ChildCount: countAllDescendants(el.Children),
		Exists:     true,
	}
}

// countAllDescendants recursively counts all descendants of an element list.
func countAllDescendants(elements []model.Element) int {
	count := len(elements)
	for _, el := range elements {
		count += countAllDescendants(el.Children)
	}
	return count
}

// stateChanged compares a pre-action snapshot against the current element tree.
// Returns true if ANY property changed (element gone, value changed, focus
// changed, selected changed, title changed, child count changed, bounds changed).
// The heuristic is deliberately broad: any change = action succeeded.
func stateChanged(pre elementSnapshot, postElements []model.Element) bool {
	postEl := findElementByID(postElements, pre.ID)
	if postEl == nil {
		return true // element disappeared (e.g. dialog dismissed, navigation)
	}
	if postEl.Title != pre.Title {
		return true
	}
	if postEl.Value != pre.Value {
		return true
	}
	if postEl.Focused != pre.Focused {
		return true
	}
	if postEl.Selected != pre.Selected {
		return true
	}
	if postEl.Bounds != pre.Bounds {
		return true
	}
	if countAllDescendants(postEl.Children) != pre.ChildCount {
		return true
	}
	return false
}

// verifyOptions holds the parameters for action verification.
type verifyOptions struct {
	Verify      bool
	VerifyDelay int // ms to wait before re-reading
	MaxRetries  int // max retry attempts (total attempts = 1 + MaxRetries)
}

// addVerifyFlags adds --verify, --verify-delay, and --max-retries flags to a command.
func addVerifyFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("verify", false, "Verify action succeeded by checking for UI changes; retry with fallback if no change detected")
	cmd.Flags().Int("verify-delay", 100, "Delay in ms before verifying (default: 100)")
	cmd.Flags().Int("max-retries", 2, "Max retry attempts when verification fails (default: 2)")
}

// getVerifyFlags reads --verify, --verify-delay, and --max-retries from a command.
func getVerifyFlags(cmd *cobra.Command) verifyOptions {
	verify, _ := cmd.Flags().GetBool("verify")
	verifyDelay, _ := cmd.Flags().GetInt("verify-delay")
	maxRetries, _ := cmd.Flags().GetInt("max-retries")
	return verifyOptions{
		Verify:      verify,
		VerifyDelay: verifyDelay,
		MaxRetries:  maxRetries,
	}
}

// verifyResult holds the outcome of a verification attempt.
type verifyResult struct {
	Verified    bool   // Was verification performed and did it pass?
	Retried     bool   // Was a retry needed?
	RetryMethod string // What fallback method was used
	RetryReason string // Why retry was needed
	// PostElements is the post-action element tree, available for reuse by --post-read.
	PostElements []model.Element
	// PostState is the formatted agent string from the post-action read.
	PostState string
}

// fallbackAction represents a retry strategy with a method name and executor.
type fallbackAction struct {
	Method  string // e.g. "action", "offset-click", "set-value", "type"
	Execute func() error
}

// verifyAction performs verification after an action: waits, re-reads the tree,
// checks if the UI changed. If not, retries with fallback strategies in order.
// Returns a verifyResult describing the outcome.
// postReadMaxElements controls the element cap for the formatted state string
// (0 = auto: 200 for web, unlimited for native).
func verifyAction(
	provider *platform.Provider,
	pre elementSnapshot,
	opts verifyOptions,
	appName, window string, windowID, pid int,
	fallbacks []fallbackAction,
	postReadMaxElements int,
) verifyResult {
	result := verifyResult{Verified: true}

	// Wait, then re-read
	postElements, postState := readPostActionElements(provider, appName, window, windowID, pid, opts.VerifyDelay, postReadMaxElements)
	result.PostElements = postElements
	result.PostState = postState

	if postElements == nil {
		// Could not read tree — can't verify, assume success (best-effort)
		return result
	}

	if stateChanged(pre, postElements) {
		return result
	}

	// No change detected — retry with fallbacks
	retriesLeft := opts.MaxRetries
	for _, fb := range fallbacks {
		if retriesLeft <= 0 {
			break
		}
		retriesLeft--

		if err := fb.Execute(); err != nil {
			continue // fallback failed, try next
		}

		// Re-read after fallback
		postElements, postState = readPostActionElements(provider, appName, window, windowID, pid, opts.VerifyDelay, postReadMaxElements)
		result.PostElements = postElements
		result.PostState = postState

		if postElements != nil && stateChanged(pre, postElements) {
			result.Retried = true
			result.RetryMethod = fb.Method
			result.RetryReason = "no UI change detected, retried with " + fb.Method
			return result
		}
	}

	// All retries exhausted, no change detected
	result.Verified = false
	if len(fallbacks) > 0 && opts.MaxRetries > 0 {
		result.Retried = true
		result.RetryMethod = fallbacks[len(fallbacks)-1].Method
		result.RetryReason = "action did not produce expected UI change after retries"
	} else {
		result.RetryReason = "action did not produce expected UI change"
	}
	return result
}

// findFocusedElementRaw searches the element tree for the deepest element
// with Focused=true, returning the raw *model.Element (not ElementInfo).
// Uses the same deepest-focused logic as findFocusedElement.
func findFocusedElementRaw(elements []model.Element) *model.Element {
	return findDeepestFocused(elements)
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}
