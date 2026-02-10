package cmd

import (
	"fmt"
	"sort"
	"testing"

	"github.com/mj1618/desktop-cli/internal/model"
)

// buildGmailTree creates a simplified accessibility tree mimicking the Gmail
// compose-over-inbox scenario described in the bug report.
//
//	root (id=1)
//	├── inbox (id=2, group)
//	│   ├── row1 (id=3, chk, title="unread, me, Test Subject, 20:59")
//	│   └── row2 (id=4, chk, title="unread, me, test subject, 20:48")
//	└── composeDialog (id=5, group)
//	    ├── to (id=6, input, desc="To")
//	    ├── subject (id=7, input, desc="Subject")  ← focused
//	    └── body (id=8, txt, desc="Body")
func buildGmailTree() []model.Element {
	return []model.Element{
		{
			ID: 1, Role: "group", Title: "Gmail",
			Children: []model.Element{
				{
					ID: 2, Role: "group", Title: "Inbox",
					Children: []model.Element{
						{ID: 3, Role: "chk", Title: "unread, me, Test Subject, 20:59"},
						{ID: 4, Role: "chk", Title: "unread, me, test subject, 20:48"},
					},
				},
				{
					ID: 5, Role: "group", Title: "Compose",
					Children: []model.Element{
						{ID: 6, Role: "input", Description: "To"},
						{ID: 7, Role: "input", Description: "Subject", Focused: true},
						{ID: 8, Role: "txt", Description: "Body"},
					},
				},
			},
		},
	}
}

func TestCollectLeafMatches_Substring(t *testing.T) {
	tree := buildGmailTree()
	matches := collectLeafMatches(tree, "subject", nil, false)
	// Substring matches: row1 (id=3), row2 (id=4), subject input (id=7)
	if len(matches) < 3 {
		t.Fatalf("expected at least 3 substring matches, got %d", len(matches))
	}
}

func TestCollectLeafMatches_Exact(t *testing.T) {
	tree := buildGmailTree()
	matches := collectLeafMatches(tree, "subject", nil, true)
	// Exact match: only id=7 (desc="Subject")
	if len(matches) != 1 {
		t.Fatalf("expected 1 exact match, got %d", len(matches))
	}
	if matches[0].ID != 7 {
		t.Fatalf("expected match id=7, got id=%d", matches[0].ID)
	}
}

func TestCollectLeafMatches_ExactWithRole(t *testing.T) {
	tree := buildGmailTree()
	roles := map[string]bool{"input": true}
	matches := collectLeafMatches(tree, "subject", roles, true)
	if len(matches) != 1 {
		t.Fatalf("expected 1 exact+role match, got %d", len(matches))
	}
	if matches[0].ID != 7 {
		t.Fatalf("expected match id=7, got id=%d", matches[0].ID)
	}
}

func TestNarrowByFocusProximity(t *testing.T) {
	tree := buildGmailTree()

	// Get all substring matches for "subject"
	matches := collectLeafMatches(tree, "subject", nil, false)
	if len(matches) < 2 {
		t.Fatalf("need multiple matches for focus test, got %d", len(matches))
	}

	narrowed := narrowByFocusProximity(tree, matches)

	// Focus is on id=7 in the compose dialog (id=5).
	// id=7 shares the deepest common ancestor (id=5→compose) with the focus.
	// id=3,4 are in the inbox (id=2), which has a shallower common ancestor (id=1).
	if len(narrowed) != 1 {
		t.Fatalf("expected focus proximity to narrow to 1, got %d", len(narrowed))
	}
	if narrowed[0].ID != 7 {
		t.Fatalf("expected narrowed match id=7, got id=%d", narrowed[0].ID)
	}
}

func TestNarrowByFocusProximity_NoFocus(t *testing.T) {
	// Tree with no focused element — should return all matches unchanged.
	tree := []model.Element{
		{ID: 1, Role: "group", Children: []model.Element{
			{ID: 2, Role: "txt", Title: "Hello"},
			{ID: 3, Role: "txt", Title: "Hello world"},
		}},
	}
	matches := collectLeafMatches(tree, "hello", nil, false)
	narrowed := narrowByFocusProximity(tree, matches)
	if len(narrowed) != len(matches) {
		t.Fatalf("expected %d matches (unchanged), got %d", len(matches), len(narrowed))
	}
}

func TestFindPathToID(t *testing.T) {
	tree := buildGmailTree()

	path := findPathToID(tree, 7)
	// root(1) → compose(5) → subject(7)
	// Note: root(1) contains inbox(2) and compose(5) as children
	expected := []int{1, 5, 7}
	if len(path) != len(expected) {
		t.Fatalf("expected path %v, got %v", expected, path)
	}
	for i, id := range expected {
		if path[i] != id {
			t.Fatalf("expected path[%d]=%d, got %d", i, id, path[i])
		}
	}
}

func TestFindPathToID_NotFound(t *testing.T) {
	tree := buildGmailTree()
	path := findPathToID(tree, 999)
	if path != nil {
		t.Fatalf("expected nil for missing ID, got %v", path)
	}
}

func TestCommonPrefixLen(t *testing.T) {
	tests := []struct {
		a, b     []int
		expected int
	}{
		{[]int{1, 5, 7}, []int{1, 5, 7}, 3},
		{[]int{1, 5, 7}, []int{1, 2, 3}, 1},
		{[]int{1, 5, 7}, []int{1, 5, 8}, 2},
		{[]int{1, 2}, []int{3, 4}, 0},
		{nil, []int{1}, 0},
		{[]int{1}, nil, 0},
	}
	for _, tt := range tests {
		got := commonPrefixLen(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("commonPrefixLen(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestScopeID(t *testing.T) {
	tree := buildGmailTree()

	// Scope to compose dialog (id=5) — should only find id=7
	scopeEl := findElementByID(tree, 5)
	if scopeEl == nil {
		t.Fatal("scope element not found")
	}
	matches := collectLeafMatches(scopeEl.Children, "subject", nil, false)
	if len(matches) != 1 {
		t.Fatalf("expected 1 scoped match, got %d", len(matches))
	}
	if matches[0].ID != 7 {
		t.Fatalf("expected scoped match id=7, got id=%d", matches[0].ID)
	}
}

func TestTextMatchesElement_Exact(t *testing.T) {
	el := model.Element{Title: "Subject", Description: "test"}

	// Exact match on title
	if !textMatchesElement(el, "subject", true) {
		t.Error("expected exact match on title 'Subject'")
	}
	// Exact should NOT match substring
	if textMatchesElement(el, "subj", true) {
		t.Error("exact match should not match substring")
	}
	// Substring should match
	if !textMatchesElement(el, "subj", false) {
		t.Error("substring match should match 'subj' in 'Subject'")
	}
}

func TestTextMatchesElement_ExactStripsShortcutSuffix(t *testing.T) {
	// Plain parenthetical suffix
	el := model.Element{Description: "Save (Ctrl+S)"}
	if !textMatchesElement(el, "save", true) {
		t.Error("exact match should match after stripping parenthetical suffix")
	}

	// Unicode directional markers around shortcut (as seen in Chrome/Gmail)
	el2 := model.Element{Description: "Send \u202a(\u2318Enter)\u202c"}
	if !textMatchesElement(el2, "send", true) {
		t.Error("exact match should match after stripping Unicode-wrapped shortcut suffix")
	}

	// Should still reject unrelated text
	if textMatchesElement(el, "sav", true) {
		t.Error("exact match should not match substring even after stripping suffix")
	}
}

// buildCalculatorTree creates a simplified accessibility tree mimicking
// Calculator where each digit appears as both a txt (display) and btn.
//
//	root (id=1)
//	├── display (id=2, txt, value="347")
//	│   ├── digitTxt3 (id=3, txt, value="3")
//	│   ├── digitTxt4 (id=4, txt, value="4")
//	│   └── digitTxt7 (id=5, txt, value="7")
//	└── keypad (id=6, group)
//	    ├── btn3 (id=7, btn, desc="3")
//	    ├── btn4 (id=8, btn, desc="4")
//	    └── btn7 (id=9, btn, desc="7")
func buildCalculatorTree() []model.Element {
	return []model.Element{
		{
			ID: 1, Role: "group", Title: "Calculator",
			Children: []model.Element{
				{
					ID: 2, Role: "txt", Value: "347",
					Children: []model.Element{
						{ID: 3, Role: "txt", Value: "3"},
						{ID: 4, Role: "txt", Value: "4"},
						{ID: 5, Role: "txt", Value: "7"},
					},
				},
				{
					ID: 6, Role: "group", Title: "Keypad",
					Children: []model.Element{
						{ID: 7, Role: "btn", Description: "3"},
						{ID: 8, Role: "btn", Description: "4"},
						{ID: 9, Role: "btn", Description: "7"},
					},
				},
			},
		},
	}
}

func TestPreferInteractiveElements_MixedRoles(t *testing.T) {
	tree := buildCalculatorTree()
	// "3" matches txt (id=3) and btn (id=7)
	matches := collectLeafMatches(tree, "3", nil, false)
	if len(matches) < 2 {
		t.Fatalf("expected at least 2 matches, got %d", len(matches))
	}

	narrowed := preferInteractiveElements(matches)
	if len(narrowed) != 1 {
		t.Fatalf("expected 1 interactive match, got %d", len(narrowed))
	}
	if narrowed[0].ID != 7 {
		t.Fatalf("expected btn id=7, got id=%d role=%s", narrowed[0].ID, narrowed[0].Role)
	}
}

func TestPreferInteractiveElements_AllSameCategory(t *testing.T) {
	// Two buttons with the same text — should NOT filter, still ambiguous
	matches := []*model.Element{
		{ID: 1, Role: "btn", Title: "Submit"},
		{ID: 2, Role: "btn", Title: "Submit"},
	}
	narrowed := preferInteractiveElements(matches)
	if len(narrowed) != 2 {
		t.Fatalf("expected 2 matches unchanged (all interactive), got %d", len(narrowed))
	}
}

func TestPreferInteractiveElements_AllStatic(t *testing.T) {
	// All static text — should NOT filter
	matches := []*model.Element{
		{ID: 1, Role: "txt", Title: "Hello"},
		{ID: 2, Role: "txt", Title: "Hello world"},
	}
	narrowed := preferInteractiveElements(matches)
	if len(narrowed) != 2 {
		t.Fatalf("expected 2 matches unchanged (all static), got %d", len(narrowed))
	}
}

func TestPreferInteractiveElements_MultipleInteractive(t *testing.T) {
	// Mix of static and multiple interactive — should keep all interactive
	matches := []*model.Element{
		{ID: 1, Role: "txt", Title: "Save"},
		{ID: 2, Role: "btn", Title: "Save"},
		{ID: 3, Role: "lnk", Title: "Save"},
	}
	narrowed := preferInteractiveElements(matches)
	if len(narrowed) != 2 {
		t.Fatalf("expected 2 interactive matches, got %d", len(narrowed))
	}
	for _, m := range narrowed {
		if m.Role == "txt" {
			t.Fatalf("static element should have been filtered out, got role=%s", m.Role)
		}
	}
}

// buildChecklistTree creates a simplified accessibility tree mimicking Apple Notes
// with a checklist where checkboxes are not exposed in the accessibility tree.
// The text items are present but not interactive — the nearest interactive elements
// are the checkboxes (simulated as buttons here since Notes doesn't expose them).
//
//	root (id=1, group)
//	├── title (id=2, txt, title="Shopping List")
//	├── checkboxRow1 (id=3, group, bounds=[500,100,300,20])
//	│   ├── checkbox1 (id=4, chk, bounds=[500,100,20,20])
//	│   └── label1 (id=5, txt, title="Buy milk", bounds=[525,100,275,20])
//	├── checkboxRow2 (id=6, group, bounds=[500,125,300,20])
//	│   ├── checkbox2 (id=7, chk, bounds=[500,125,20,20])
//	│   └── label2 (id=8, txt, title="Buy bread", bounds=[525,125,275,20])
//	└── checkboxRow3 (id=9, group, bounds=[500,150,300,20])
//	    ├── checkbox3 (id=10, chk, bounds=[500,150,20,20])
//	    └── label3 (id=11, txt, title="Buy eggs", bounds=[525,150,275,20])
func buildChecklistTree() []model.Element {
	return []model.Element{
		{
			ID: 1, Role: "group", Title: "Notes",
			Children: []model.Element{
				{ID: 2, Role: "txt", Title: "Shopping List", Bounds: [4]int{500, 60, 300, 20}},
				{
					ID: 3, Role: "group", Bounds: [4]int{500, 100, 300, 20},
					Children: []model.Element{
						{ID: 4, Role: "chk", Bounds: [4]int{500, 100, 20, 20}},
						{ID: 5, Role: "txt", Title: "Buy milk", Bounds: [4]int{525, 100, 275, 20}},
					},
				},
				{
					ID: 6, Role: "group", Bounds: [4]int{500, 125, 300, 20},
					Children: []model.Element{
						{ID: 7, Role: "chk", Bounds: [4]int{500, 125, 20, 20}},
						{ID: 8, Role: "txt", Title: "Buy bread", Bounds: [4]int{525, 125, 275, 20}},
					},
				},
				{
					ID: 9, Role: "group", Bounds: [4]int{500, 150, 300, 20},
					Children: []model.Element{
						{ID: 10, Role: "chk", Bounds: [4]int{500, 150, 20, 20}},
						{ID: 11, Role: "txt", Title: "Buy eggs", Bounds: [4]int{525, 150, 275, 20}},
					},
				},
			},
		},
	}
}

func TestFindNearestInteractiveElement_ChecklistCheckbox(t *testing.T) {
	tree := buildChecklistTree()

	// "Buy milk" label is at (525,100,275,20), center = (662,110)
	// checkbox1 is at (500,100,20,20), center = (510,110) — same row, 152px away
	// checkbox2 is at (500,125,20,20), center = (510,135) — next row, ~155px away
	label := findElementByID(tree, 5) // "Buy milk"
	if label == nil {
		t.Fatal("label not found")
	}

	nearest := findNearestInteractiveElement(tree, label, "")
	if nearest == nil {
		t.Fatal("expected to find nearest interactive element")
	}
	if nearest.ID != 4 {
		t.Fatalf("expected nearest to be checkbox1 (id=4), got id=%d role=%s", nearest.ID, nearest.Role)
	}
}

func TestFindNearestInteractiveElement_SecondRow(t *testing.T) {
	tree := buildChecklistTree()

	label := findElementByID(tree, 8) // "Buy bread"
	if label == nil {
		t.Fatal("label not found")
	}

	nearest := findNearestInteractiveElement(tree, label, "")
	if nearest == nil {
		t.Fatal("expected to find nearest interactive element")
	}
	if nearest.ID != 7 {
		t.Fatalf("expected nearest to be checkbox2 (id=7), got id=%d role=%s", nearest.ID, nearest.Role)
	}
}

func TestFindNearestInteractiveElement_NoInteractive(t *testing.T) {
	// Tree with only static elements
	tree := []model.Element{
		{ID: 1, Role: "txt", Title: "Hello", Bounds: [4]int{0, 0, 100, 20}},
		{ID: 2, Role: "txt", Title: "World", Bounds: [4]int{0, 25, 100, 20}},
	}
	nearest := findNearestInteractiveElement(tree, &tree[0], "")
	if nearest != nil {
		t.Fatalf("expected nil when no interactive elements exist, got id=%d", nearest.ID)
	}
}

func TestFindNearestInteractiveElement_TooFar(t *testing.T) {
	// Interactive element exists but is beyond nearMaxRadius
	tree := []model.Element{
		{ID: 1, Role: "txt", Title: "Label", Bounds: [4]int{100, 100, 100, 20}},
		{ID: 2, Role: "btn", Title: "Far away", Bounds: [4]int{900, 900, 100, 30}},
	}
	nearest := findNearestInteractiveElement(tree, &tree[0], "")
	if nearest != nil {
		t.Fatalf("expected nil when interactive element is beyond radius, got id=%d", nearest.ID)
	}
}

func TestFindNearestInteractiveElement_DirectionLeft(t *testing.T) {
	tree := buildChecklistTree()
	label := findElementByID(tree, 5) // "Buy milk"
	if label == nil {
		t.Fatal("label not found")
	}

	nearest := findNearestInteractiveElement(tree, label, "left")
	if nearest == nil {
		t.Fatal("expected to find nearest interactive element to the left")
	}
	if nearest.ID != 4 {
		t.Fatalf("expected checkbox1 (id=4) to the left, got id=%d", nearest.ID)
	}
}

func TestFindNearestInteractiveElement_DirectionRight(t *testing.T) {
	// Button to the right of a label, checkbox to the left
	tree := []model.Element{
		{ID: 1, Role: "chk", Bounds: [4]int{100, 100, 20, 20}},       // left, center=(110,110)
		{ID: 2, Role: "txt", Title: "Label", Bounds: [4]int{130, 100, 80, 20}}, // center=(170,110)
		{ID: 3, Role: "btn", Title: "Edit", Bounds: [4]int{220, 100, 40, 20}},  // right, center=(240,110)
	}
	label := &tree[1]

	// direction=right should find the button (id=3), not the checkbox (id=1)
	nearest := findNearestInteractiveElement(tree, label, "right")
	if nearest == nil {
		t.Fatal("expected to find element to the right")
	}
	if nearest.ID != 3 {
		t.Fatalf("expected btn (id=3) to the right, got id=%d", nearest.ID)
	}
}

func TestNearFallbackOffset(t *testing.T) {
	anchor := &model.Element{Bounds: [4]int{200, 100, 100, 20}}

	// Default (left): 20px left of left edge, vertically centered
	x, y := nearFallbackOffset(anchor, "")
	if x != 180 || y != 110 {
		t.Errorf("default fallback: expected (180,110), got (%d,%d)", x, y)
	}

	// Explicit left
	x, y = nearFallbackOffset(anchor, "left")
	if x != 180 || y != 110 {
		t.Errorf("left fallback: expected (180,110), got (%d,%d)", x, y)
	}

	// Right: 20px right of right edge
	x, y = nearFallbackOffset(anchor, "right")
	if x != 320 || y != 110 {
		t.Errorf("right fallback: expected (320,110), got (%d,%d)", x, y)
	}

	// Above: 20px above top edge
	x, y = nearFallbackOffset(anchor, "above")
	if x != 250 || y != 80 {
		t.Errorf("above fallback: expected (250,80), got (%d,%d)", x, y)
	}

	// Below: 20px below bottom edge
	x, y = nearFallbackOffset(anchor, "below")
	if x != 250 || y != 140 {
		t.Errorf("below fallback: expected (250,140), got (%d,%d)", x, y)
	}
}

func TestIsDisplayElement(t *testing.T) {
	// txt with value and no press action → display
	el := model.Element{Role: "txt", Value: "123"}
	if !isDisplayElement(el) {
		t.Error("expected txt with value to be a display element")
	}

	// txt with press action → NOT display (it's a button-like element)
	el2 := model.Element{Role: "txt", Value: "123", Actions: []string{"press"}}
	if isDisplayElement(el2) {
		t.Error("expected txt with press action to NOT be a display element")
	}

	// txt without value → NOT display
	el3 := model.Element{Role: "txt", Title: "Label"}
	if isDisplayElement(el3) {
		t.Error("expected txt without value to NOT be a display element")
	}

	// btn with value → NOT display (wrong role)
	el4 := model.Element{Role: "btn", Value: "123"}
	if isDisplayElement(el4) {
		t.Error("expected btn to NOT be a display element")
	}
}

func TestCollectDisplayElements_Primary(t *testing.T) {
	// Simulate Calculator with expression (small) and result (large)
	tree := []model.Element{
		{
			ID: 1, Role: "group", Title: "Calculator",
			Children: []model.Element{
				{ID: 9, Role: "txt", Value: "347×29+156", Bounds: [4]int{624, 731, 124, 26}},
				{ID: 11, Role: "txt", Value: "10219", Bounds: [4]int{664, 761, 83, 36}},
			},
		},
	}
	displays := collectDisplayElements(tree)
	if len(displays) != 2 {
		t.Fatalf("expected 2 display elements, got %d", len(displays))
	}

	// Convert to infos and apply primary logic (same as readDisplayElements)
	infos := make([]ElementInfo, len(displays))
	for i, el := range displays {
		infos[i] = *elementInfoFromElement(el)
	}
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

	// The result element (id=11, height=36) should be primary
	for _, info := range infos {
		if info.ID == 11 {
			if !info.Primary {
				t.Error("expected result element (id=11, height=36) to be primary")
			}
		} else if info.ID == 9 {
			if info.Primary {
				t.Error("expected expression element (id=9, height=26) to NOT be primary")
			}
		}
	}
}

func TestCollectDisplayElements_SingleNoPrimary(t *testing.T) {
	// Single display element → no primary marking needed
	tree := []model.Element{
		{ID: 1, Role: "txt", Value: "42", Bounds: [4]int{100, 100, 50, 30}},
	}
	displays := collectDisplayElements(tree)
	if len(displays) != 1 {
		t.Fatalf("expected 1 display element, got %d", len(displays))
	}

	infos := make([]ElementInfo, len(displays))
	for i, el := range displays {
		infos[i] = *elementInfoFromElement(el)
	}
	// With only 1 element, primary should not be set
	if infos[0].Primary {
		t.Error("single display element should not have primary set")
	}
}

func TestMaxDisplayElements(t *testing.T) {
	// Verify that maxDisplayElements constant is defined and reasonable
	if maxDisplayElements < 1 || maxDisplayElements > 100 {
		t.Fatalf("maxDisplayElements should be between 1 and 100, got %d", maxDisplayElements)
	}

	// Build a tree with more display elements than the cap
	var children []model.Element
	for i := 0; i < maxDisplayElements+10; i++ {
		children = append(children, model.Element{
			ID:     i + 1,
			Role:   "txt",
			Value:  fmt.Sprintf("item %d", i),
			Bounds: [4]int{0, i * 20, 100, 18},
		})
	}
	tree := []model.Element{{ID: 0, Role: "group", Children: children}}

	displays := collectDisplayElements(tree)
	if len(displays) != maxDisplayElements+10 {
		t.Fatalf("expected %d display elements from collect, got %d", maxDisplayElements+10, len(displays))
	}

	// Simulate the cap applied in readDisplayElements
	if len(displays) > maxDisplayElements {
		displays = displays[:maxDisplayElements]
	}
	if len(displays) != maxDisplayElements {
		t.Fatalf("expected capped to %d, got %d", maxDisplayElements, len(displays))
	}
}

func TestDistanceToBounds(t *testing.T) {
	// Element at (100,100,20,20) center=(110,110), target at (200,200,20,20) center=(210,210)
	el := &model.Element{Bounds: [4]int{100, 100, 20, 20}}
	target := [4]int{200, 200, 20, 20}
	d := distanceToBounds(el, target)
	// Distance between (110,110) and (210,210) = sqrt(100^2 + 100^2) ≈ 141.42
	if d < 141 || d > 142 {
		t.Errorf("expected distance ~141.42, got %f", d)
	}

	// Same position → distance 0
	el2 := &model.Element{Bounds: [4]int{200, 200, 20, 20}}
	d2 := distanceToBounds(el2, target)
	if d2 != 0 {
		t.Errorf("expected distance 0, got %f", d2)
	}
}

func TestDisplayElementsProximitySorting(t *testing.T) {
	// Simulate a window with sidebar elements (left, y=0..500) and content elements (right, y=300..500).
	// Target is in the content area at (800,400,100,20).
	// Without proximity sorting, the first 20 would all be sidebar items.
	// With proximity sorting, content-area items should be included.

	var sidebarElements []model.Element
	for i := 0; i < 25; i++ {
		sidebarElements = append(sidebarElements, model.Element{
			ID:     i + 1,
			Role:   "txt",
			Value:  fmt.Sprintf("sidebar %d", i),
			Bounds: [4]int{50, i * 20, 150, 18}, // left side
		})
	}
	var contentElements []model.Element
	for i := 0; i < 10; i++ {
		contentElements = append(contentElements, model.Element{
			ID:     100 + i,
			Role:   "txt",
			Value:  fmt.Sprintf("content %d", i),
			Bounds: [4]int{800, 300 + i*20, 200, 18}, // right side, near target
		})
	}

	tree := []model.Element{
		{ID: 0, Role: "group", Children: sidebarElements},
		{ID: 50, Role: "group", Children: contentElements},
	}

	displays := collectDisplayElements(tree)
	if len(displays) != 35 {
		t.Fatalf("expected 35 display elements, got %d", len(displays))
	}

	// Sort by proximity to a target in the content area
	targetBounds := [4]int{800, 400, 100, 20}
	sort.Slice(displays, func(i, j int) bool {
		return distanceToBounds(displays[i], targetBounds) < distanceToBounds(displays[j], targetBounds)
	})

	// Cap to maxDisplayElements
	if len(displays) > maxDisplayElements {
		displays = displays[:maxDisplayElements]
	}

	// All 10 content elements should be in the result (they are closer to target)
	contentCount := 0
	for _, el := range displays {
		if el.ID >= 100 {
			contentCount++
		}
	}
	if contentCount != 10 {
		t.Errorf("expected all 10 content elements in result, got %d", contentCount)
	}
}

// buildFlightsTree creates a simplified accessibility tree mimicking the
// Chrome "Flights" search result scenario where many zero-height elements match.
//
//	root (id=1, window)
//	├── visible link (id=2, lnk, desc="Flights", bounds 68x48)
//	├── visible link (id=3, lnk, desc="Sydney To London Flights...", bounds 228x31)
//	├── zero-height link (id=4, lnk, desc="Show flights on Google Flights", bounds 652x0)
//	├── zero-height link (id=5, lnk, desc="Cheap Flights from Sydney...", bounds 460x0)
//	├── zero-height link (id=6, lnk, desc="Find Cheap Flights...", bounds 493x0)
//	├── zero-width link (id=7, lnk, desc="Flights deal", bounds 0x31)
//	└── visible link (id=8, lnk, desc="Sydney to London Flights guide", bounds 400x31)
func buildFlightsTree() []model.Element {
	return []model.Element{
		{
			ID: 1, Role: "window", Title: "Google Chrome",
			Children: []model.Element{
				{ID: 2, Role: "lnk", Description: "Flights", Bounds: [4]int{417, 214, 68, 48}},
				{ID: 3, Role: "lnk", Description: "Sydney To London Flights are cheap", Bounds: [4]int{310, 832, 228, 31}},
				{ID: 4, Role: "lnk", Description: "Show flights on Google Flights", Bounds: [4]int{310, 1117, 652, 0}},
				{ID: 5, Role: "lnk", Description: "Cheap Flights from Sydney to London", Bounds: [4]int{310, 1117, 460, 0}},
				{ID: 6, Role: "lnk", Description: "Find Cheap Flights to London", Bounds: [4]int{310, 1117, 493, 0}},
				{ID: 7, Role: "lnk", Description: "Flights deal", Bounds: [4]int{310, 1117, 0, 31}},
				{ID: 8, Role: "lnk", Description: "Sydney to London Flights guide", Bounds: [4]int{310, 978, 400, 31}},
			},
		},
	}
}

func TestFilterVisibleElements_RemovesZeroDimension(t *testing.T) {
	tree := buildFlightsTree()
	// "flights" matches all 7 links (id=2..8)
	roles := map[string]bool{"lnk": true}
	matches := collectLeafMatches(tree, "flights", roles, false)
	if len(matches) != 7 {
		t.Fatalf("expected 7 total matches, got %d", len(matches))
	}

	visible := filterVisibleElements(matches)
	// Should exclude id=4 (height=0), id=5 (height=0), id=6 (height=0), id=7 (width=0)
	if len(visible) != 3 {
		t.Fatalf("expected 3 visible matches, got %d", len(visible))
	}
	for _, m := range visible {
		if m.Bounds[2] == 0 || m.Bounds[3] == 0 {
			t.Errorf("zero-dimension element should not be in visible results: id=%d bounds=(%d,%d,%d,%d)",
				m.ID, m.Bounds[0], m.Bounds[1], m.Bounds[2], m.Bounds[3])
		}
	}
}

func TestFilterVisibleElements_AllVisible(t *testing.T) {
	matches := []*model.Element{
		{ID: 1, Role: "btn", Bounds: [4]int{0, 0, 100, 30}},
		{ID: 2, Role: "btn", Bounds: [4]int{0, 40, 100, 30}},
	}
	visible := filterVisibleElements(matches)
	if len(visible) != 2 {
		t.Fatalf("expected 2 visible matches, got %d", len(visible))
	}
}

func TestFilterVisibleElements_AllZero(t *testing.T) {
	matches := []*model.Element{
		{ID: 1, Role: "lnk", Bounds: [4]int{0, 0, 0, 30}},
		{ID: 2, Role: "lnk", Bounds: [4]int{0, 0, 100, 0}},
	}
	visible := filterVisibleElements(matches)
	if len(visible) != 0 {
		t.Fatalf("expected 0 visible matches, got %d", len(visible))
	}
}

func TestFilterVisibleElements_Empty(t *testing.T) {
	visible := filterVisibleElements(nil)
	if len(visible) != 0 {
		t.Fatalf("expected 0 visible matches for nil input, got %d", len(visible))
	}
}

// --- Verification helper tests ---

func TestCountAllDescendants(t *testing.T) {
	// Empty
	if n := countAllDescendants(nil); n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
	// Flat
	elems := []model.Element{{ID: 1}, {ID: 2}, {ID: 3}}
	if n := countAllDescendants(elems); n != 3 {
		t.Errorf("expected 3, got %d", n)
	}
	// Nested: 2 top-level + 1 child + 1 grandchild = 4
	nested := []model.Element{
		{ID: 1, Children: []model.Element{
			{ID: 2, Children: []model.Element{
				{ID: 3},
			}},
		}},
		{ID: 4},
	}
	if n := countAllDescendants(nested); n != 4 {
		t.Errorf("expected 4, got %d", n)
	}
}

func TestSnapshotElement(t *testing.T) {
	el := model.Element{
		ID: 42, Title: "Submit", Value: "val", Focused: true, Selected: false,
		Bounds:   [4]int{10, 20, 100, 30},
		Children: []model.Element{{ID: 43}, {ID: 44}},
	}
	snap := snapshotElement(&el)
	if snap.ID != 42 {
		t.Errorf("ID: expected 42, got %d", snap.ID)
	}
	if snap.Title != "Submit" {
		t.Errorf("Title: expected 'Submit', got %q", snap.Title)
	}
	if snap.Value != "val" {
		t.Errorf("Value: expected 'val', got %q", snap.Value)
	}
	if !snap.Focused {
		t.Error("expected Focused=true")
	}
	if snap.Selected {
		t.Error("expected Selected=false")
	}
	if snap.Bounds != [4]int{10, 20, 100, 30} {
		t.Errorf("Bounds: unexpected %v", snap.Bounds)
	}
	if snap.ChildCount != 2 {
		t.Errorf("ChildCount: expected 2, got %d", snap.ChildCount)
	}
	if !snap.Exists {
		t.Error("expected Exists=true")
	}
}

func TestStateChanged_NoChange(t *testing.T) {
	el := model.Element{
		ID: 1, Title: "OK", Value: "v", Focused: true, Selected: false,
		Bounds:   [4]int{0, 0, 50, 20},
		Children: []model.Element{{ID: 2}},
	}
	pre := snapshotElement(&el)
	// Post tree is identical
	postTree := []model.Element{el}
	if stateChanged(pre, postTree) {
		t.Error("expected no change, got change")
	}
}

func TestStateChanged_ElementDisappeared(t *testing.T) {
	el := model.Element{ID: 1, Title: "Dialog"}
	pre := snapshotElement(&el)
	// Post tree has a different element
	postTree := []model.Element{{ID: 99, Title: "Other"}}
	if !stateChanged(pre, postTree) {
		t.Error("expected change (element disappeared), got no change")
	}
}

func TestStateChanged_ValueChanged(t *testing.T) {
	el := model.Element{ID: 1, Value: "old"}
	pre := snapshotElement(&el)
	postTree := []model.Element{{ID: 1, Value: "new"}}
	if !stateChanged(pre, postTree) {
		t.Error("expected change (value changed), got no change")
	}
}

func TestStateChanged_TitleChanged(t *testing.T) {
	el := model.Element{ID: 1, Title: "Submit"}
	pre := snapshotElement(&el)
	postTree := []model.Element{{ID: 1, Title: "Submitted"}}
	if !stateChanged(pre, postTree) {
		t.Error("expected change (title changed), got no change")
	}
}

func TestStateChanged_FocusChanged(t *testing.T) {
	el := model.Element{ID: 1, Focused: false}
	pre := snapshotElement(&el)
	postTree := []model.Element{{ID: 1, Focused: true}}
	if !stateChanged(pre, postTree) {
		t.Error("expected change (focus changed), got no change")
	}
}

func TestStateChanged_SelectedChanged(t *testing.T) {
	el := model.Element{ID: 1, Selected: false}
	pre := snapshotElement(&el)
	postTree := []model.Element{{ID: 1, Selected: true}}
	if !stateChanged(pre, postTree) {
		t.Error("expected change (selected changed), got no change")
	}
}

func TestStateChanged_BoundsChanged(t *testing.T) {
	el := model.Element{ID: 1, Bounds: [4]int{0, 0, 50, 20}}
	pre := snapshotElement(&el)
	postTree := []model.Element{{ID: 1, Bounds: [4]int{10, 10, 50, 20}}}
	if !stateChanged(pre, postTree) {
		t.Error("expected change (bounds changed), got no change")
	}
}

func TestStateChanged_ChildCountChanged(t *testing.T) {
	el := model.Element{ID: 1, Children: []model.Element{{ID: 2}}}
	pre := snapshotElement(&el)
	// Post tree: element now has 2 children
	postTree := []model.Element{{ID: 1, Children: []model.Element{{ID: 2}, {ID: 3}}}}
	if !stateChanged(pre, postTree) {
		t.Error("expected change (child count changed), got no change")
	}
}

func TestFindFocusedElementRaw(t *testing.T) {
	tree := []model.Element{
		{ID: 1, Role: "window", Children: []model.Element{
			{ID: 2, Role: "btn", Title: "OK"},
			{ID: 3, Role: "input", Title: "Name", Focused: true},
		}},
	}
	focused := findFocusedElementRaw(tree)
	if focused == nil {
		t.Fatal("expected to find focused element")
	}
	if focused.ID != 3 {
		t.Errorf("expected focused element ID=3, got %d", focused.ID)
	}

	// No focused element
	tree2 := []model.Element{{ID: 1, Role: "btn"}}
	if findFocusedElementRaw(tree2) != nil {
		t.Error("expected nil when no element is focused")
	}
}

func TestFindDeepestFocused_MultipleFocusedElements(t *testing.T) {
	// Simulates Reminders where a parent cell and the actual input both have Focused=true.
	// The deepest input should be returned.
	tree := []model.Element{
		{ID: 1, Role: "window", Children: []model.Element{
			{ID: 9, Role: "cell", Description: "My Lists", Focused: true, Children: []model.Element{
				{ID: 14, Role: "cell", Description: "Reminders", Focused: true},
			}},
			{ID: 20, Role: "group", Children: []model.Element{
				{ID: 50, Role: "input", Value: "Test List", Focused: true},
			}},
		}},
	}
	focused := findDeepestFocused(tree)
	if focused == nil {
		t.Fatal("expected to find focused element")
	}
	// Should prefer the input (id=50) over the cells (id=9, id=14)
	if focused.ID != 50 {
		t.Errorf("expected deepest focused to be input id=50, got id=%d role=%s", focused.ID, focused.Role)
	}
}

func TestFindDeepestFocused_PrefersInputRole(t *testing.T) {
	// Two sibling focused elements at the same depth: one input, one cell.
	// The input should be preferred.
	tree := []model.Element{
		{ID: 1, Role: "window", Children: []model.Element{
			{ID: 2, Role: "cell", Title: "Some cell", Focused: true},
			{ID: 3, Role: "input", Title: "Name field", Focused: true},
		}},
	}
	focused := findDeepestFocused(tree)
	if focused == nil {
		t.Fatal("expected to find focused element")
	}
	if focused.ID != 3 {
		t.Errorf("expected input id=3 to be preferred, got id=%d role=%s", focused.ID, focused.Role)
	}
}

func TestFindDeepestFocused_SingleFocused(t *testing.T) {
	// Single focused element should be returned as before.
	tree := []model.Element{
		{ID: 1, Role: "window", Children: []model.Element{
			{ID: 2, Role: "btn", Title: "OK"},
			{ID: 3, Role: "input", Title: "Name", Focused: true},
		}},
	}
	focused := findDeepestFocused(tree)
	if focused == nil {
		t.Fatal("expected to find focused element")
	}
	if focused.ID != 3 {
		t.Errorf("expected focused id=3, got id=%d", focused.ID)
	}
}

func TestFindFocusedElement_DeepestReturned(t *testing.T) {
	// Verify the public findFocusedElement also returns the deepest element
	tree := []model.Element{
		{ID: 1, Role: "window", Focused: true, Children: []model.Element{
			{ID: 2, Role: "group", Focused: true, Children: []model.Element{
				{ID: 3, Role: "input", Title: "Search", Focused: true},
			}},
		}},
	}
	focused := findFocusedElement(tree)
	if focused == nil {
		t.Fatal("expected to find focused element")
	}
	if focused.ID != 3 {
		t.Errorf("expected deepest focused id=3, got id=%d", focused.ID)
	}
}

// buildNotesMultiPaneTree creates a simplified accessibility tree mimicking
// Apple Notes with a sidebar (note preview list) and main content area, where
// the same text "Buy groceries" appears in both the sidebar preview and the
// active note's checklist.
//
//	root (id=1, window)
//	├── sidebar (id=2, group, bounds=[0,0,300,800])
//	│   ├── notePreview1 (id=3, txt, title="My Task List\n9:45pm\nBuy groceries", bounds=[20,100,260,60])
//	│   └── notePreview2 (id=4, txt, title="Other Note", bounds=[20,170,260,40])
//	└── content (id=5, group, bounds=[300,0,700,800])
//	    ├── noteTitle (id=6, txt, title="My Task List", bounds=[320,50,660,30])
//	    ├── checkRow1 (id=7, group, bounds=[320,100,660,20])
//	    │   ├── checkbox1 (id=8, chk, bounds=[320,100,20,20])
//	    │   └── label1 (id=9, txt, title="Buy groceries", bounds=[345,100,300,20])
//	    └── checkRow2 (id=10, group, bounds=[320,130,660,20])
//	        ├── checkbox2 (id=11, chk, bounds=[320,130,20,20])
//	        └── label2 (id=12, txt, title="Walk the dog", bounds=[345,130,300,20])
func buildNotesMultiPaneTree() []model.Element {
	return []model.Element{
		{
			ID: 1, Role: "window", Title: "Notes", Bounds: [4]int{0, 0, 1000, 800},
			Children: []model.Element{
				{
					ID: 2, Role: "group", Title: "Sidebar", Bounds: [4]int{0, 0, 300, 800},
					Children: []model.Element{
						{ID: 3, Role: "txt", Title: "My Task List\n9:45pm\nBuy groceries", Bounds: [4]int{20, 100, 260, 60}},
						{ID: 4, Role: "txt", Title: "Other Note", Bounds: [4]int{20, 170, 260, 40}},
					},
				},
				{
					ID: 5, Role: "group", Title: "Content", Bounds: [4]int{300, 0, 700, 800},
					Children: []model.Element{
						{ID: 6, Role: "txt", Title: "My Task List", Bounds: [4]int{320, 50, 660, 30}},
						{
							ID: 7, Role: "group", Bounds: [4]int{320, 100, 660, 20},
							Children: []model.Element{
								{ID: 8, Role: "chk", Bounds: [4]int{320, 100, 20, 20}},
								{ID: 9, Role: "txt", Title: "Buy groceries", Bounds: [4]int{345, 100, 300, 20}},
							},
						},
						{
							ID: 10, Role: "group", Bounds: [4]int{320, 130, 660, 20},
							Children: []model.Element{
								{ID: 11, Role: "chk", Bounds: [4]int{320, 130, 20, 20}},
								{ID: 12, Role: "txt", Title: "Walk the dog", Bounds: [4]int{345, 130, 300, 20}},
							},
						},
					},
				},
			},
		},
	}
}

func TestPickBestNearMatch_PrefersContentArea(t *testing.T) {
	tree := buildNotesMultiPaneTree()

	// "Buy groceries" matches:
	// - id=3 (sidebar preview, x=20) — the wrong one
	// - id=9 (content area checklist, x=345) — the correct one
	matches := filterVisibleElements(collectLeafMatches(tree, "buy groceries", nil, false))
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}

	best := pickBestNearMatch(tree, matches)
	if best.ID != 9 {
		t.Fatalf("expected content-area match id=9, got id=%d (x=%d)", best.ID, best.Bounds[0])
	}
}

func TestPickBestNearMatch_ContentAreaCheckbox(t *testing.T) {
	tree := buildNotesMultiPaneTree()

	// Full flow: text match + --near should find the checkbox in the content area
	matches := filterVisibleElements(collectLeafMatches(tree, "buy groceries", nil, false))
	best := pickBestNearMatch(tree, matches)

	// best should be id=9 (content area "Buy groceries" label at x=345)
	if best.ID != 9 {
		t.Fatalf("expected content-area match id=9, got id=%d", best.ID)
	}

	// Then findNearestInteractiveElement from that match should find checkbox1 (id=8)
	nearest := findNearestInteractiveElement(tree, best, "")
	if nearest == nil {
		t.Fatal("expected to find nearest interactive element")
	}
	if nearest.ID != 8 {
		t.Fatalf("expected checkbox id=8 near content-area text, got id=%d", nearest.ID)
	}
}

func TestPickBestNearMatch_SingleMatch(t *testing.T) {
	tree := buildNotesMultiPaneTree()

	// "Walk the dog" only appears once — should return it directly
	matches := filterVisibleElements(collectLeafMatches(tree, "walk the dog", nil, false))
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	best := pickBestNearMatch(tree, matches)
	if best.ID != 12 {
		t.Fatalf("expected id=12, got id=%d", best.ID)
	}
}

func TestPickBestNearMatch_SameXUsesFocusProximity(t *testing.T) {
	// Two matches at the same X position — should fall back to focus proximity
	tree := []model.Element{
		{
			ID: 1, Role: "window",
			Children: []model.Element{
				{
					ID: 2, Role: "group",
					Children: []model.Element{
						{ID: 3, Role: "txt", Title: "Buy groceries", Bounds: [4]int{300, 100, 200, 20}},
					},
				},
				{
					ID: 4, Role: "group",
					Children: []model.Element{
						{ID: 5, Role: "input", Focused: true, Bounds: [4]int{300, 200, 200, 20}},
						{ID: 6, Role: "txt", Title: "Buy groceries", Bounds: [4]int{300, 230, 200, 20}},
					},
				},
			},
		},
	}
	matches := filterVisibleElements(collectLeafMatches(tree, "buy groceries", nil, false))
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}

	best := pickBestNearMatch(tree, matches)
	// id=6 is in the same group as the focused element, so should be preferred
	if best.ID != 6 {
		t.Fatalf("expected id=6 (near focused element), got id=%d", best.ID)
	}
}

func TestBoolPtr(t *testing.T) {
	p := boolPtr(true)
	if p == nil || !*p {
		t.Error("expected *true")
	}
	p = boolPtr(false)
	if p == nil || *p {
		t.Error("expected *false")
	}
}
