package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ProductSchema returns the rosetta schema that describes a Product
func ProductSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"productId":         schema.String{Format: "objectId"},
			"userId":            schema.String{Format: "objectId"},
			"merchantAccountId": schema.String{Format: "objectId"},
			"remoteId":          schema.String{Format: "text", MaxLength: 256},
			"name":              schema.String{Format: "text", MaxLength: 256},
			"price":             schema.String{Format: "text", MaxLength: 64},
			"icon":              schema.String{Format: "text", MaxLength: 64},
			"adminHref":         schema.String{Format: "url"},
		},
	}
}

// GetStringOK returns the named property. Implements schema.StringGetter.
func (product Product) GetStringOK(property string) (string, bool) {
	switch property {

	case "productId":
		return product.ProductID.Hex(), true

	case "userId":
		return product.UserID.Hex(), true

	case "merchantAccountId":
		return product.MerchantAccountID.Hex(), true

	case "remoteId":
		return product.RemoteID, true

	case "name":
		return product.Name, true

	case "price":
		return product.Price, true

	case "icon":
		return product.Icon, true

	case "adminHref":
		return product.AdminHref, true
	}

	return "", false
}

// SetString writes the named property. Implements schema.StringSetter.
func (product *Product) SetString(property string, value string) bool {
	switch property {

	case "productId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			product.ProductID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			product.UserID = objectID
			return true
		}

	case "merchantAccountId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			product.MerchantAccountID = objectID
			return true
		}

	case "remoteId":
		product.RemoteID = value
		return true

	case "name":
		product.Name = value
		return true

	case "price":
		product.Price = value
		return true

	case "icon":
		product.Icon = value
		return true

	case "adminHref":
		product.AdminHref = value
		return true

	}

	return false
}
