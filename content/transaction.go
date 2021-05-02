package content

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

type Transaction interface {
	Execute(*Content) error
	Description() string
}

func ParseTransaction(in map[string]interface{}) (Transaction, error) {

	data := datatype.Map(in)

	switch data.GetString("type") {

	case "new-item":
		return NewItemTransaction{
			ParentID:   data.GetInt("parentId"),
			ChildIndex: data.GetInt("childIndex"),
			Type:       data.GetString("type"),
		}, nil

	case "update-item":

		return UpdateItemTransaction{
			ItemID: data.GetInt("itemId"),
			Data:   in,
			Hash:   data.GetString("hash"),
		}, nil

	case "delete-item":

		return DeleteItemTransaction{
			ItemID: data.GetInt("itemId"),
			Hash:   data.GetString("hash"),
		}, nil

	case "move-item":

		return MoveItemTransaction{
			ItemID:      data.GetInt("itemId"),
			NewParentID: data.GetInt("newParentId"),
			Position:    data.GetInt("position"),
		}, nil
	}

	return NilTransaction(data), derp.New(500, "content.ParseTransaction", "Invalid Transaction", in)
}
