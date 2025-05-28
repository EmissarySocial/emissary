package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PrivilegeSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"privilegeId":       schema.String{Format: "objectId"},
			"identityId":        schema.String{Format: "objectId"},
			"userId":            schema.String{Format: "objectId"},
			"circleId":          schema.String{Format: "objectId", Required: false},
			"merchantAccountId": schema.String{Format: "objectId", Required: false},
			"remotePersonId":    schema.String{MaxLength: 256},
			"remoteProductId":   schema.String{MaxLength: 256},
			"remotePurchaseId":  schema.String{MaxLength: 256},
		},
	}
}

/*********************************
 * Getter/Setter Interfaces
 *********************************/

func (privilege *Privilege) GetStringOK(name string) (string, bool) {
	switch name {

	case "privilegeId":
		return privilege.PrivilegeID.Hex(), true

	case "identityId":
		return privilege.IdentityID.Hex(), true

	case "userId":
		return privilege.UserID.Hex(), true

	case "circleId":
		return privilege.CircleID.Hex(), true

	case "merchantAccountId":
		return privilege.MerchantAccountID.Hex(), true
	}

	return "", false
}

func (privilege *Privilege) SetString(name string, value string) bool {

	switch name {

	case "privilegeId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			privilege.PrivilegeID = objectID
			return true
		}

	case "identityId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			privilege.IdentityID = objectID
			return true
		}
	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			privilege.UserID = objectID
			return true
		}

	case "circleId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			privilege.CircleID = objectID
			return true
		}

	case "merchantAccountId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			privilege.MerchantAccountID = objectID
			return true
		}
	}

	return false
}

func (privilege *Privilege) GetPointer(name string) (any, bool) {

	switch name {

	case "remotePersonId":
		return &privilege.RemotePersonID, true

	case "remoteProductId":
		return &privilege.RemoteProductID, true

	case "remotePurchaseId":
		return &privilege.RemotePurchaseID, true
	}

	return nil, false
}
