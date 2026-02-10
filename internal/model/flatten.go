package model

// FlatElement is an element with a path breadcrumb instead of children.
type FlatElement struct {
	ID          int      `yaml:"i"            json:"i"`
	Role        string   `yaml:"r"            json:"r"`
	Subrole     string   `yaml:"sr,omitempty" json:"sr,omitempty"`
	Title       string   `yaml:"t,omitempty"  json:"t,omitempty"`
	Value       string   `yaml:"v,omitempty"  json:"v,omitempty"`
	Description string   `yaml:"d,omitempty"  json:"d,omitempty"`
	Bounds      [4]int   `yaml:"b"            json:"b"`
	Focused     bool     `yaml:"f,omitempty"  json:"f,omitempty"`
	Enabled     *bool    `yaml:"e,omitempty"  json:"e,omitempty"`
	Selected    bool     `yaml:"s,omitempty"  json:"s,omitempty"`
	Actions     []string `yaml:"a,omitempty"  json:"a,omitempty"`
	Ref         string   `yaml:"ref,omitempty" json:"ref,omitempty"`
	Path        string   `yaml:"p,omitempty"  json:"p,omitempty"`
}

// FlattenElements converts a tree of elements into a flat list.
// Each element gets a path string showing its location in the tree
// using abbreviated role names joined with " > ".
func FlattenElements(elements []Element) []FlatElement {
	var result []FlatElement
	for _, el := range elements {
		flattenRecursive(el, "", &result)
	}
	return result
}

func flattenRecursive(el Element, parentPath string, result *[]FlatElement) {
	currentPath := el.Role
	if parentPath != "" {
		currentPath = parentPath + " > " + el.Role
	}

	flat := FlatElement{
		ID:          el.ID,
		Role:        el.Role,
		Subrole:     el.Subrole,
		Title:       el.Title,
		Value:       el.Value,
		Description: el.Description,
		Bounds:      el.Bounds,
		Focused:     el.Focused,
		Enabled:     el.Enabled,
		Selected:    el.Selected,
		Actions:     el.Actions,
		Ref:         el.Ref,
		Path:        currentPath,
	}
	*result = append(*result, flat)

	for _, child := range el.Children {
		flattenRecursive(child, currentPath, result)
	}
}
