package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ProductSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"productId":         schema.String{Format: "objectId"},
			"merchantAccountId": schema.String{Format: "objectId"},
			"remoteId":          schema.String{MaxLength: 1024},
			"name":              schema.String{MaxLength: 64},
			"description":       schema.String{MaxLength: 256},
			"price":             schema.String{MaxLength: 32},
			"recurringType":     schema.String{MaxLength: 32, Enum: []string{ProductRecurringTypeOnetime, ProductRecurringTypeWeekly, ProductRecurringTypeMonthly, ProductRecurringTypeYearly}},
			"isFeatured":        schema.Boolean{},
		},
	}
}

/*********************************
 * Getter/Setter Interfaces
 *********************************/

func (product *Product) GetPointer(name string) (any, bool) {

	switch name {

	case "remoteId":
		return &product.RemoteID, true

	case "name":
		return &product.Name, true

	case "description":
		return &product.Description, true

	case "price":
		return &product.Price, true

	case "recurringType":
		return &product.RecurringType, true

	case "isFeatured":
		return &product.IsFeatured, true

	default:
		return nil, false
	}
}

func (product *Product) GetStringOK(name string) (string, bool) {
	switch name {

	case "productId":
		return product.ProductID.Hex(), true

	case "merchantAccountId":
		return product.MerchantAccountID.Hex(), true

	default:
		return "", false
	}
}

func (product *Product) SetString(name string, value string) bool {

	switch name {

	case "productId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			product.ProductID = objectID
			return true
		}

	case "merchantAccountId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			product.MerchantAccountID = objectID
			return true
		}
	}

	return false
}
