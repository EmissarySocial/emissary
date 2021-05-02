package content

import "github.com/benpate/derp"

type NewItemTransaction struct {
	ParentID   int    `json:"parentId"   form:"parentId"`   // ID of the parent node that will contain this new item
	ChildIndex int    `json:"childIndex" form:"childIndex"` // Where to place the new child in the parent's list of children
	Type       string `json:"type"       form:"type"`       // What kind of widget to create
	Hash       string `json:"hash"       form:"hash"`
}

func (txn NewItemTransaction) Execute(content *Content) error {

	item := Item{
		Type: txn.Type,
	}

	_, err := content.AddReference(txn.ParentID, item, txn.Hash)

	return derp.Wrap(err, "content.NewItemTransaction.Execute", "Error adding new item")
}

func (txn NewItemTransaction) Description() string {
	return "New Item (" + txn.Type + ")"
}
