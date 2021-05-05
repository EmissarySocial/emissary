package content

import (
	"github.com/benpate/derp"
)

// Content represents a complete package of content
type Content []Item

func (content Content) Render(library Library) string {
	return library.Render(content, 0)
}

// DeleteReference removes an item from a parent
func (content Content) DeleteReference(parentID int, deleteID int, check string) error {

	// Bounds check
	if (parentID < 0) || (parentID >= len(content)) {
		return derp.New(500, "content.Create", "Parent index out of bounds", parentID, deleteID)
	}

	// Bounds check
	if (deleteID < 0) || (deleteID >= len(content)) {
		return derp.New(500, "content.Create", "Child index out of bounds", parentID, deleteID)
	}

	// validate checksum
	if check != content[parentID].Check {
		return derp.New(derp.CodeForbiddenError, "content.Create", "Invalid Checksum")
	}

	// Remove the references to the deleted Item
	for _, childID := range content[deleteID].Refs {
		content.DeleteReference(deleteID, childID, content[deleteID].Check)
	}

	// Remove teh deleted item
	content[deleteID] = Item{}

	// Remove the deleted item from the parent's list of child references
	content[parentID].DeleteReference(deleteID)

	// Success!
	return nil
}

func (content *Content) Compact() {
	front := 0
	back := len(*content) - 1

	for front < back {

		if (*content)[front].Type != "" {
			front = front + 1
			continue
		}

		if (*content)[back].Type == "" {
			back = back - 1
			continue
		}

		content.move(back, front)
	}

	if (*content)[back].Type != "" {
		back = back - 1
	}

	*content = (*content)[:back]
}

// move physically moves an item from one index to another (overwriting the target location)
// and updates references
func (content Content) move(from int, to int) {

	content[to] = content[from]
	content[from] = Item{}

	for index := range content {
		content[index].UpdateReference(from, to)
	}
}
