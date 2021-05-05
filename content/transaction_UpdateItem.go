package content

import (
	"github.com/benpate/derp"
)

type UpdateItemTransaction struct {
	ItemID int                    `json:"itemId" form:"itemId"`
	Data   map[string]interface{} `json:"data"   form:"data"`
	Check  string                 `json:"hash"   form:"hash"`
}

func (txn UpdateItemTransaction) Execute(content *Content) error {

	// Bounds check
	if (txn.ItemID < 0) || (txn.ItemID >= len(*content)) {
		return derp.New(500, "content.UpdateItemTransaction", "Index out of bounds", txn.ItemID)
	}

	// Validate checksum
	if txn.Check != (*content)[txn.ItemID].Check {
		return derp.New(derp.CodeForbiddenError, "content.UpdateItemTransaction", "Invalid Checksum")
	}

	// Update data
	(*content)[txn.ItemID].Data = txn.Data
	return nil
}

func (txn UpdateItemTransaction) Description() string {
	return "Update Item"
}
