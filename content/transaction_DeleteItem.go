package content

type DeleteItemTransaction struct {
	ItemID int    `json:"itemId" form:"itemId"`
	Check  string `json:"check"  form:"check"`
}

func (txn DeleteItemTransaction) Execute(content *Content) error {
	return nil
}

func (txn DeleteItemTransaction) Description() string {
	return "Delete Item"
}
