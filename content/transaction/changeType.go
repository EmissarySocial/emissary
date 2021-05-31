package transaction

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/content"
)

type ChangeType struct {
	ItemID   int    `json:"itemId"   form:"itemId"`   // ID of the root item that will be added to
	ItemType string `json:"itemType" form:"itemType"` // Type of content item to add
	Check    string `json:"check"    form:"check"`    // Checksum to validation transaction.
}

// Execute performs the ChangeType transaction on the provided content structure
func (txn ChangeType) Execute(c *content.Content) error {

	// Bounds check
	if (txn.ItemID < 0) || (txn.ItemID >= len(*c)) {
		return derp.New(500, "content.transaction.ChangeType", "Index out of bounds", txn)
	}

	// Hash check
	if txn.Check != (*c)[txn.ItemID].Check {
		return derp.New(derp.CodeForbiddenError, "content.transaction.ChangeType", "Invalid Checksum")
	}

	(*c)[txn.ItemID].Type = txn.ItemType
	(*c)[txn.ItemID].Data = datatype.Map{}

	return nil
}

func (txn ChangeType) Description() string {
	return "Change Item Type (" + txn.ItemType + ")"
}
