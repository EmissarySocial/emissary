package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ProductSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"productId":         schema.String{},
			"userId":            schema.String{},
			"merchantAccountId": schema.String{},
			"remoteId":          schema.String{},
			"name":              schema.String{},
			"price":             schema.String{},
			"icon":              schema.String{},
			"adminHref":         schema.String{},
		},
	}
}

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
