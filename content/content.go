package content

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// Content represents a complete package of content
type Content []Item

func (content Content) Render(library Library) string {
	return library.Render(content, 0)
}

// AddReference adds a new content item to the content section
func (content *Content) AddReference(parentID int, item Item, hash string) (int, error) {

	// Bounds check
	if (parentID < 0) || (parentID >= len(*content)) {
		return 0, derp.New(500, "content.Create", "Index out of bounds", parentID, item)
	}

	// Hash check
	if hash != (*content)[parentID].Hash {
		return 0, derp.New(derp.CodeForbiddenError, "content.Create", "Invalid Hash Value")
	}

	// Reset the Hash for each new item
	item.NewHash()

	// Add the new item to the content container.
	newID := len(*content)
	*content = append(*content, item)

	// Add a reference to the new item in the parent.
	(*content)[parentID].AddReference(newID)

	// Success!
	return newID, nil
}

// DeleteReference removes an item from a parent
func (content Content) DeleteReference(parentID int, deleteID int, hash string) error {

	// Bounds check
	if (parentID < 0) || (parentID >= len(content)) {
		return derp.New(500, "content.Create", "Parent index out of bounds", parentID, deleteID)
	}

	// Bounds check
	if (deleteID < 0) || (deleteID >= len(content)) {
		return derp.New(500, "content.Create", "Child index out of bounds", parentID, deleteID)
	}

	// Hash check
	if hash != content[parentID].Hash {
		return derp.New(derp.CodeForbiddenError, "content.Create", "Invalid Hash Value")
	}

	// Remove the references to the deleted Item
	for _, childID := range content[deleteID].Refs {
		content.DeleteReference(deleteID, childID, content[deleteID].Hash)
	}

	// Remove teh deleted item
	content[deleteID] = Item{}

	// Remove the deleted item from the parent's list of child references
	content[parentID].DeleteReference(deleteID)

	// Success!
	return nil
}

// UpdateItem updates the content of an item in place.
func (content Content) UpdateItem(itemID int, data datatype.Map, hash string) error {

	// Bounds check
	if (itemID < 0) || (itemID >= len(content)) {
		return derp.New(500, "content.Create", "Index out of bounds", itemID)
	}

	// Hash check
	if hash != content[itemID].Hash {
		return derp.New(derp.CodeForbiddenError, "content.Create", "Invalid Hash Value")
	}

	// Update data
	content[itemID].Data = data
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
