package cmd

import "github.com/mj1618/desktop-cli/internal/model"

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
