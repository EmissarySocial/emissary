package transaction

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/content"
)

const newItemPositionBefore = 0

const newItemPositionAfter = 1

type NewItem struct {
	ItemID int    `json:"itemId" form:"itemId"` // ID of the root item that will be added to
	Place  string `json:"place"  form:"place"`  // ABOVE, BELOW, LEFT, RIGHT
	Type   string `json:"type"   form:"type"`   // Type of content item to add
	Check  string `json:"check"  form:"check"`  // Checksum to validation transaction.
}

// Execute performs the NewItem transaction on the provided content structure
func (txn NewItem) Execute(c *content.Content) error {

	// Bounds check
	if (txn.ItemID < 0) || (txn.ItemID >= len(*c)) {
		return derp.New(500, "content.transaction.NewItem", "Index out of bounds", txn)
	}

	/* // Hash check
	if txn.Check != (*c)[txn.ItemID].Check {
		return derp.New(derp.CodeForbiddenError, "content.transaction.NewItem", "Invalid Checksum")
	}
	*/

	// Create a new item to insert into the content
	newItem := content.Item{
		Type:  content.ItemTypeWYSIWYG,
		Check: content.NewChecksum(),
	}

	sibling := (*c)[txn.ItemID]

	// Insert at head or tail of a container
	if sibling.Type == content.ItemTypeContainer {
		switch txn.Place {
		case content.ContainerPlaceAbove:
			if sibling.GetString("style") == content.ContainerStyleRows {
				addFirstRef(c, txn.ItemID, newItem)
				return nil
			}

		case content.ContainerPlaceBelow:
			if sibling.GetString("style") == content.ContainerStyleRows {
				addLastRef(c, txn.ItemID, newItem)
				return nil
			}

		case content.ContainerPlaceLeft:
			if sibling.GetString("style") == content.ContainerStyleColumns {
				addFirstRef(c, txn.ItemID, newItem)
				return nil
			}

		case content.ContainerPlaceRight:
			if sibling.GetString("style") == content.ContainerStyleColumns {
				addLastRef(c, txn.ItemID, newItem)
				return nil
			}
		}
	}

	// Locate the parent
	parentIndex, parent := findParent(c, txn.ItemID)

	// If the parent is already a container (of the right direction) then
	// we only need to add this new item into it...
	if parent != nil && parent.Type == content.ItemTypeContainer {

		switch txn.Place {
		case content.ContainerPlaceAbove:
			if parent.GetString("style") == content.ContainerStyleRows {
				insertRef(c, parentIndex, txn.ItemID, newItem, newItemPositionBefore)
				return nil
			}

		case content.ContainerPlaceBelow:
			if parent.GetString("style") == content.ContainerStyleRows {
				insertRef(c, parentIndex, txn.ItemID, newItem, newItemPositionAfter)
				return nil
			}

		case content.ContainerPlaceLeft:
			if parent.GetString("style") == content.ContainerStyleColumns {
				insertRef(c, parentIndex, txn.ItemID, newItem, newItemPositionBefore)
				return nil
			}

		case content.ContainerPlaceRight:
			if parent.GetString("style") == content.ContainerStyleColumns {
				insertRef(c, parentIndex, txn.ItemID, newItem, newItemPositionAfter)
				return nil
			}
		}
	}

	// Fall through means that we need to make a new container.
	// ABOVE,BELOW require a ROWS container
	// LEFT,RIGHT require a COLUMNS container

	switch txn.Place {
	case content.ContainerPlaceAbove:
		replaceWithContainer(c, content.ContainerStyleRows, txn.ItemID, newItem, newItemPositionBefore)
		return nil

	case content.ContainerPlaceBelow:
		replaceWithContainer(c, content.ContainerStyleRows, txn.ItemID, newItem, newItemPositionAfter)
		return nil

	case content.ContainerPlaceLeft:
		replaceWithContainer(c, content.ContainerStyleColumns, txn.ItemID, newItem, newItemPositionBefore)
		return nil

	case content.ContainerPlaceRight:
		replaceWithContainer(c, content.ContainerStyleColumns, txn.ItemID, newItem, newItemPositionAfter)
		return nil
	}

	// Something bad happened.  Abort! Abort!
	return derp.New(500, "content.transaction.NewItem", "Invalid transaction", txn)

}

func (txn NewItem) Description() string {
	return "New Item (" + txn.Type + ")"
}

func addFirstRef(c *content.Content, parentID int, newItem content.Item) {
	newID := c.AddItem(newItem)
	oldRefs := (*c)[parentID].Refs
	(*c)[parentID].Refs = append([]int{newID}, oldRefs...)
}

func addLastRef(c *content.Content, parentID int, newItem content.Item) {
	newID := c.AddItem(newItem)
	(*c)[parentID].Refs = append((*c)[parentID].Refs, newID)
}

// insertRef inserts `newItem` into the content, and places a reference to it inside of the
// `parentID` item, either BEFORE or AFTER the `childID` item
func insertRef(c *content.Content, parentID int, childID int, newItem content.Item, position int) {
	newID := c.AddItem(newItem)
	newRefs := make([]int, 0)

	for _, itemID := range (*c)[parentID].Refs {
		if itemID == childID {
			if position == newItemPositionBefore {
				newRefs = append(newRefs, newID, itemID)
			} else {
				newRefs = append(newRefs, itemID, newID)
			}
		} else {
			newRefs = append(newRefs, itemID)
		}
	}

	(*c)[parentID].Refs = newRefs
}

// replaceWithContainer replaces an existing content Item with a container (of a specific style),
// moves the original item to the end of the content structure,
// then inserts the new item into correct position of the new container (either BEFORE or AFTER the original Item)
func replaceWithContainer(c *content.Content, style string, itemID int, newItem content.Item, position int) {

	// reset the checksum on the current item
	(*c)[itemID].NewChecksum()

	// copy the current item to the end of the content structure
	newLocationID := len(*c)
	*c = append(*c, (*c)[itemID])

	// insert a container in the spot where the original content was
	(*c)[itemID] = content.Item{
		Type:  content.ItemTypeContainer,
		Refs:  []int{newLocationID},
		Check: content.NewChecksum(),
		Data: datatype.Map{
			"style": style,
		},
	}

	// insert a reference to the newItem into the new container
	insertRef(c, itemID, newLocationID, newItem, position)
}
