package transaction

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/content"
)

type DeleteItem struct {
	ItemID int    `json:"itemId" form:"itemId"`
	Check  string `json:"check"  form:"check"`
}

func (txn DeleteItem) Execute(c *content.Content) error {

	// Find parent index and record
	parentID, parent := c.GetParent(txn.ItemID)

	// Remove parent's reference to this item
	parent.DeleteReference(txn.ItemID)

	// Recursively delete this item and all of its children
	return deleteItem(c, parentID, txn.ItemID, txn.Check)
}

func (txn DeleteItem) Description() string {
	return "Delete Item"
}

// DeleteReference removes an item from a parent
func deleteItem(c *content.Content, parentID int, deleteID int, check string) error {

	// Bounds check
	if (parentID < 0) || (parentID >= len(*c)) {
		return derp.New(500, "content.Create", "Parent index out of bounds", parentID, deleteID)
	}

	// Bounds check
	if (deleteID < 0) || (deleteID >= len(*c)) {
		return derp.New(500, "content.Create", "Child index out of bounds", parentID, deleteID)
	}

	// validate checksum
	if check != (*c)[parentID].Check {
		return derp.New(derp.CodeForbiddenError, "content.Create", "Invalid Checksum")
	}

	// Remove all children from the content
	if len((*c)[deleteID].Refs) > 0 {
		childCheck := (*c)[deleteID].Check
		for _, childID := range (*c)[deleteID].Refs {
			deleteItem(c, deleteID, childID, childCheck)
		}
	}

	// Remove the deleted item
	(*c)[deleteID] = content.Item{}

	// Success!
	return nil
}
