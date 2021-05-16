package transaction

import "github.com/benpate/ghost/content"

type MoveItem struct {
	ItemID      int    `json:"itemId"      form:"itemId"`
	NewParentID int    `json:"newParentId" form:"newParentId"`
	Position    int    `json:"position"    form:"position"`
	Check       string `json:"check"       form:"check"`
}

func (txn MoveItem) Execute(c *content.Content) error {
	return nil
}

func (txn MoveItem) Description() string {
	return "Move Item"
}
