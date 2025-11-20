package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ImportSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"importId": schema.String{Format: "objectId", Required: true},
			"userId":   schema.String{Format: "objectId", Required: true},
			"stateId": schema.String{
				Enum: []string{
					ImportStateNew,
					ImportStateDoAuthorize,
					ImportStateAuthorizing,
					ImportStateAuthorizationError,
					ImportStateAuthorized,
					ImportStateDoImport,
					ImportStateImporting,
					ImportStateImportError,
					ImportStateDoMove,
					ImportStateMoving,
					ImportStateMoveError,
					ImportStateDone,
				},
				Required: true},
			"sourceId":         schema.String{},
			"stateDescription": schema.String{},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

func (record *Import) GetPointer(name string) (any, bool) {

	switch name {

	case "stateId":
		return &record.StateID, true

	case "sourceId":
		return &record.SourceID, true

	case "stateDescription":
		return &record.StateDescription, true
	}

	return nil, false
}

func (record Import) GetStringOK(name string) (string, bool) {

	switch name {

	case "importId":
		return record.ImportID.Hex(), true

	case "userId":
		return record.UserID.Hex(), true
	}

	return "", false
}

func (record *Import) SetString(name string, value string) bool {

	switch name {

	case "importId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			record.ImportID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			record.UserID = objectID
			return true
		}
	}

	return false
}
