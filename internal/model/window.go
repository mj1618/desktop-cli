package model

// Window represents an application window.
type Window struct {
	App     string `json:"app"`
	PID     int    `json:"pid"`
	Title   string `json:"title"`
	ID      int    `json:"id"`
	Bounds  [4]int `json:"bounds"`
	Focused bool   `json:"focused,omitempty"`
}
