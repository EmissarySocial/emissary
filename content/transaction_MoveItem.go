package content

type MoveItemTransaction struct {
	ItemID      int    `json:"itemId"      form:"itemId"`
	NewParentID int    `json:"newParentId" form:"newParentId"`
	Position    int    `json:"position"    form:"position"`
	Hash        string `json:"hash"        form:"hash"`
}

func (txn MoveItemTransaction) Execute(content *Content) error {
	return nil
}

func (txn MoveItemTransaction) Description() string {
	return "Move Item"
}
