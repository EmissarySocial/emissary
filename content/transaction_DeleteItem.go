package content

type DeleteItemTransaction struct {
	ItemID int    `json:"itemId" form:"itemId"`
	Hash   string `json:"hash"   form:"hash"`
}

func (txn DeleteItemTransaction) Execute(content *Content) error {
	return nil
}

func (txn DeleteItemTransaction) Description() string {
	return "Delete Item"
}
