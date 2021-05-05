package content

import "github.com/benpate/derp"

type NewItemTransaction struct {
	ParentID   int    `json:"parentId"   form:"parentId"`   // ID of the parent node that will contain this new item
	ChildIndex int    `json:"childIndex" form:"childIndex"` // Where to place the new child in the parent's list of children
	ItemType   string `json:"itemType"   form:"itemType"`   // What kind of widget to create
	Check      string `json:"check"       form:"check"`
}

func (txn NewItemTransaction) Execute(content *Content) error {

	item := Item{
		Type: txn.ItemType,
	}

	// Bounds check
	if (txn.ParentID < 0) || (txn.ParentID >= len(*content)) {
		return derp.New(500, "content.Create", "Index out of bounds", txn.ParentID, item)
	}

	// Hash check
	if txn.Check != (*content)[txn.ParentID].Check {
		return derp.New(derp.CodeForbiddenError, "content.Create", "Invalid Checksum")
	}

	// Reset the Hash for each new item
	item.NewChecksum()

	// Add the new item to the content container.

	newID := len(*content)
	*content = append((*content), item)

	// Add a reference to the new item in the parent.
	(*content)[txn.ParentID].AddReference(newID, txn.ChildIndex)

	// Success!
	return nil
}

func (txn NewItemTransaction) Description() string {
	return "New Item (" + txn.ItemType + ")"
}
