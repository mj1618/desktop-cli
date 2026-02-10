package model

// RoleMap maps macOS AXRole values to compact role codes.
var RoleMap = map[string]string{
	"AXButton":      "btn",
	"AXStaticText":  "txt",
	"AXLink":        "lnk",
	"AXImage":       "img",
	"AXTextField":   "input",
	"AXTextArea":    "input",
	"AXCheckBox":    "chk",
	"AXSwitch":      "toggle",
	"AXRadioButton": "radio",
	"AXMenu":        "menu",
	"AXMenuBar":     "menu",
	"AXMenuItem":    "menuitem",
	"AXTabGroup":    "tab",
	"AXList":        "list",
	"AXTable":       "list",
	"AXRow":         "row",
	"AXCell":        "cell",
	"AXGroup":       "group",
	"AXSplitGroup":  "group",
	"AXScrollArea":  "scroll",
	"AXToolbar":     "toolbar",
	"AXWebArea":     "web",
	"AXWindow":      "window",
}

// MetaRoles maps meta-role names to the concrete roles they expand to.
// For example, "interactive" matches roles that are likely to accept user input,
// including web app fields that Chrome exposes as "other" instead of "input".
var MetaRoles = map[string][]string{
	"interactive": {"input", "other", "chk", "toggle", "radio", "list"},
}

// ExpandRoles expands any meta-roles in the given list to their concrete roles.
// Non-meta roles are passed through unchanged. Duplicates are removed.
func ExpandRoles(roles []string) []string {
	seen := make(map[string]bool, len(roles))
	var expanded []string
	for _, r := range roles {
		if concrete, ok := MetaRoles[r]; ok {
			for _, c := range concrete {
				if !seen[c] {
					seen[c] = true
					expanded = append(expanded, c)
				}
			}
		} else if !seen[r] {
			seen[r] = true
			expanded = append(expanded, r)
		}
	}
	return expanded
}

// MapRole converts a raw accessibility role to a compact code.
func MapRole(axRole string) string {
	if short, ok := RoleMap[axRole]; ok {
		return short
	}
	return "other"
}
