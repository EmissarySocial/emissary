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
			ItemType:   data.GetString("itemType"),
			Check:      data.GetString("check"),
		}, nil

	case "update-item":

		return UpdateItemTransaction{
			ItemID: data.GetInt("itemId"),
			Data:   extractData(in),
			Check:  data.GetString("check"),
		}, nil

	case "delete-item":

		return DeleteItemTransaction{
			ItemID: data.GetInt("itemId"),
			Check:  data.GetString("check"),
		}, nil

	case "move-item":

		return MoveItemTransaction{
			ItemID:      data.GetInt("itemId"),
			NewParentID: data.GetInt("newParentId"),
			Position:    data.GetInt("position"),
			Check:       data.GetString("check"),
		}, nil
	}

	return NilTransaction(data), derp.New(500, "content.ParseTransaction", "Invalid Transaction", in)
}

func extractData(input map[string]interface{}) map[string]interface{} {

	result := make(map[string]interface{})

	for key, value := range input {
		switch key {
		case "type", "hash", "refs":
			continue
		default:
			result[key] = value
		}
	}

	return result
}
