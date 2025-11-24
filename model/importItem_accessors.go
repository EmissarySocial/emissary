package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ImportItemSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"importItemId": schema.String{Format: "objectId", Required: true},
			"importId":     schema.String{Format: "objectId", Required: true},
			"userId":       schema.String{Format: "objectId", Required: true},
			"localId":      schema.String{Format: "objectId", Required: true},
			"type":         schema.String{Required: true},
			"url":          schema.String{Required: true},
			"stateId":      schema.String{Required: true},
			"message":      schema.String{},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

func (item *ImportItem) GetPointer(name string) (any, bool) {

	switch name {

	case "type":
		return &item.Type, true

	case "url":
		return &item.URL, true

	case "stateId":
		return &item.StateID, true

	case "message":
		return &item.Message, true
	}

	return nil, false
}

func (item ImportItem) GetStringOK(name string) (string, bool) {

	switch name {

	case "importItemId":
		return item.ImportItemID.Hex(), true

	case "importId":
		return item.ImportID.Hex(), true

	case "userId":
		return item.UserID.Hex(), true

	case "localId":
		return item.LocalID.Hex(), true
	}

	return "", false
}

func (item *ImportItem) SetString(name string, value string) bool {

	switch name {

	case "importItemId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			item.ImportItemID = objectID
			return true
		}

	case "importId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			item.ImportID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			item.UserID = objectID
			return true
		}

	case "localId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			item.LocalID = objectID
			return true
		}
	}

	return false
}
