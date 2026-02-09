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

// MapRole converts a raw accessibility role to a compact code.
func MapRole(axRole string) string {
	if short, ok := RoleMap[axRole]; ok {
		return short
	}
	return "other"
}
