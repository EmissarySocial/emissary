package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PurchaseSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"purchaseId": schema.String{Format: "objectId"},
			"guestId":    schema.String{Format: "objectId"},
			"productId":  schema.String{Format: "objectId"},
			"userId":     schema.String{Format: "objectId"},
		},
	}
}

/*********************************
 * Getter/Setter Interfaces
 *********************************/

func (purchase *Purchase) GetStringOK(name string) (string, bool) {
	switch name {

	case "purchaseId":
		return purchase.PurchaseID.Hex(), true

	case "productId":
		return purchase.ProductID.Hex(), true

	case "guestId":
		return purchase.GuestID.Hex(), true

	case "userId":
		return purchase.UserID.Hex(), true

	default:
		return "", false
	}
}

func (purchase *Purchase) SetString(name string, value string) bool {

	switch name {

	case "purchaseId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			purchase.PurchaseID = objectID
			return true
		}

	case "productId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			purchase.ProductID = objectID
			return true
		}

	case "guestId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			purchase.GuestID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			purchase.UserID = objectID
			return true
		}
	}

	return false
}
