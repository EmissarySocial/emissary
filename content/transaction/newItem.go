package transaction

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/content"
)

type NewItem struct {
	ParentID   int    `json:"parentId"   form:"parentId"`   // ID of the parent node that will contain this new item
	ChildIndex int    `json:"childIndex" form:"childIndex"` // Where to place the new child in the parent's list of children
	ItemType   string `json:"itemType"   form:"itemType"`   // What kind of widget to create
	Style      string `json:"style"      form:"style"`      // Custom Style to use
	Check      string `json:"check"      form:"check"`      // Checksum to validation transaction.
}

func (txn NewItem) Execute(c *content.Content) error {

	item := content.Item{
		Type: txn.ItemType,
	}

	// Bounds check
	if (txn.ParentID < 0) || (txn.ParentID >= len(*c)) {
		return derp.New(500, "content.Create", "Index out of bounds", txn.ParentID, item)
	}

	// Hash check
	if txn.Check != (*c)[txn.ParentID].Check {
		return derp.New(derp.CodeForbiddenError, "content.Create", "Invalid Checksum")
	}

	// Reset the Hash for each new item
	item.NewChecksum()

	// Optional parameters
	if txn.Style != "" {
		item.Data["style"] = txn.Style
	}

	// Add the new item to the content container.

	newID := len(*c)
	*c = append((*c), item)

	// Add a reference to the new item in the parent.
	(*c)[txn.ParentID].AddReference(newID, txn.ChildIndex)

	// Success!
	return nil
}

func (txn NewItem) Description() string {
	return "New Item (" + txn.ItemType + ")"
}

func newItem(c *content.Content, items ...content.Item) {

	// Append items to the content slice
	*c = append(*c, items...)

	// Recursively append any of THEIR default children...
	for index := range items {
		widget := c.Widget(index)
		if defaulter, ok := widget.(content.DefaultChildrener); ok {
			children := defaulter.DefaultChildren()
			newItem(c, children...)
		}
	}
}
