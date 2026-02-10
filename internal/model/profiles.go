package model

// HasWebContent checks if the element tree contains web content
// by looking for the "web" role (AXWebArea). This indicates a browser
// or web view, where pruning empty groups is almost always desirable.
func HasWebContent(elements []Element) bool {
	for i := range elements {
		if elements[i].Role == "web" {
			return true
		}
		if HasWebContent(elements[i].Children) {
			return true
		}
	}
	return false
}

// ExpandRolesForWeb auto-expands a role list to include "other" when "input"
// is present and the element tree contains web content. Chrome exposes some
// web input fields as role "other" instead of "input".
func ExpandRolesForWeb(roles []string, hasWeb bool) ([]string, bool) {
	if !hasWeb {
		return roles, false
	}
	hasInput := false
	hasOther := false
	for _, r := range roles {
		if r == "input" {
			hasInput = true
		}
		if r == "other" {
			hasOther = true
		}
	}
	if hasInput && !hasOther {
		return append(roles, "other"), true
	}
	return roles, false
}
