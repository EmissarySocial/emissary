package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MerchantAccountSchema returns a JSON Schema that describes this object
func MerchantAccountSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"merchantAccountId": schema.String{Format: "objectId"},
			"userId":            schema.String{Format: "objectId"},
			"type":              schema.String{Enum: []string{ConnectionProviderStripe, ConnectionProviderStripeConnect}}, // ConnectionProviderPayPal,
			"name":              schema.String{MaxLength: 128},
			"description":       schema.String{MaxLength: 1024},
			"vault":             schema.Object{Wildcard: schema.String{}},
			"liveMode":          schema.Boolean{},
		},
	}
}

/******************************************
 * Getter/Setter Methods
 ******************************************/

func (merchantAccount *MerchantAccount) GetPointer(name string) (any, bool) {
	switch name {

	case "type":
		return &merchantAccount.Type, true

	case "name":
		return &merchantAccount.Name, true

	case "description":
		return &merchantAccount.Description, true

	case "vault":
		return &merchantAccount.Vault, true

	case "liveMode":
		return &merchantAccount.LiveMode, true

	default:
		return nil, false
	}
}

func (merchantAccount *MerchantAccount) GetStringOK(name string) (string, bool) {

	switch name {

	case "merchantAccountId":
		return merchantAccount.MerchantAccountID.Hex(), true

	case "userId":
		return merchantAccount.UserID.Hex(), true

	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

func (merchantAccount *MerchantAccount) SetString(name string, value string) bool {

	switch name {

	case "merchantAccountId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			merchantAccount.MerchantAccountID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			merchantAccount.UserID = objectID
			return true
		}
	}

	return false
}
