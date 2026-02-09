package model

// Element represents a UI element in the accessibility tree.
type Element struct {
	ID          int       `json:"i"`                    // Sequential integer ID
	Role        string    `json:"r"`                    // Abbreviated role code
	Title       string    `json:"t,omitempty"`          // Visible label / title
	Value       string    `json:"v,omitempty"`          // Current value
	Description string    `json:"d,omitempty"`          // Accessibility description
	Bounds      [4]int    `json:"b"`                    // [x, y, width, height]
	Focused     bool      `json:"f,omitempty"`          // Has keyboard focus
	Enabled     *bool     `json:"e,omitempty"`          // nil or true = enabled (omit); false = disabled (include)
	Selected    bool      `json:"s,omitempty"`          // Is selected
	Children    []Element `json:"c,omitempty"`          // Child elements
	Actions     []string  `json:"a,omitempty"`          // Available actions
}
