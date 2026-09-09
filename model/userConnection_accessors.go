package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserConnectionSchema returns a JSON Schema that describes this object
func UserConnectionSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"userConnectionId": schema.String{Format: "objectId"},
			"userId":           schema.String{Format: "objectId"},
			"type":             schema.String{Enum: []string{UserConnectionTypeMailchimp}},
			"isActive":         schema.Boolean{},
			"status":           schema.String{Required: true, Enum: []string{UserConnectionStatusPending, UserConnectionStatusReady, UserConnectionStatusReconnect}}, // Required, or an empty string skips the enum and reads as no state at all
			"data":             schema.Object{Wildcard: schema.String{MaxLength: 1024}},
			"vault":            schema.Object{Wildcard: schema.String{Format: "unsafe-any", MaxLength: 8192}}, // vault holds secrets; unsafe-any avoids the no-html default corrupting stored values.
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (userConnection *UserConnection) GetPointer(name string) (any, bool) {

	switch name {

	case "type":
		return &userConnection.Type, true

	case "status":
		return &userConnection.Status, true

	case "data":
		return &userConnection.Data, true

	case "vault":
		return &userConnection.Vault, true

	default:
		return nil, false
	}
}

// GetBoolOK returns the named property. Implements schema.BoolGetter.
func (userConnection *UserConnection) GetBoolOK(name string) (bool, bool) {

	if name == "isActive" {
		return userConnection.IsActive.Value(), true
	}

	return false, false
}

// GetStringOK returns the named property. Implements schema.StringGetter.
func (userConnection *UserConnection) GetStringOK(name string) (string, bool) {

	switch name {

	case "userConnectionId":
		return userConnection.UserConnectionID.Hex(), true

	case "userId":
		return userConnection.UserID.Hex(), true
	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

// SetBool writes the named property. Implements schema.BoolSetter.
func (userConnection *UserConnection) SetBool(name string, value bool) bool {

	// RULE: this goes through delta.Bool.Set, which is what lets the service see that the
	// switch moved. Assigning the field directly would lose the change.
	if name == "isActive" {
		userConnection.IsActive.Set(value)
		return true
	}

	return false
}

// SetString writes the named property. Implements schema.StringSetter.
func (userConnection *UserConnection) SetString(name string, value string) bool {

	switch name {

	case "userConnectionId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			userConnection.UserConnectionID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			userConnection.UserID = objectID
			return true
		}
	}

	return false
}
