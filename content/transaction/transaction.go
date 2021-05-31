package transaction

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/content"
)

type Transaction interface {
	Execute(*content.Content) error
	Description() string
}

func Parse(in map[string]interface{}) (Transaction, error) {

	data := datatype.Map(in)

	switch data.GetString("type") {

	case "change-type":

		return ChangeType{
			ItemID:   data.GetInt("itemId"),
			ItemType: data.GetString("itemType"),
			Check:    data.GetString("check"),
		}, nil

	case "new-item":

		return NewItem{
			ItemID:   data.GetInt("itemId"),
			Place:    data.GetString("place"),
			ItemType: data.GetString("itemType"),
			Check:    data.GetString("check"),
		}, nil

	case "update-item":

		return UpdateItem{
			ItemID: data.GetInt("itemId"),
			Data:   extractData(in),
			Check:  data.GetString("check"),
		}, nil

	case "delete-item":

		return DeleteItem{
			ItemID: data.GetInt("itemId"),
			Check:  data.GetString("check"),
		}, nil

	case "move-item":

		return MoveItem{
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
