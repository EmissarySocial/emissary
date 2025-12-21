package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ImportSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"importId":  schema.String{Format: "objectId", Required: true},
			"userId":    schema.String{Format: "objectId", Required: true},
			"sourceId":  schema.String{},
			"sourceUrl": schema.String{Format: "uri"},
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
					ImportStateReviewing,
					ImportStateDoMove,
					ImportStateDone,
				},
				Required: true},
			"message":       schema.String{},
			"totalItems":    schema.Integer{},
			"completeItems": schema.Integer{},
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

	case "sourceUrl":
		return &record.SourceURL, true

	case "message":
		return &record.Message, true

	case "totalItems":
		return &record.TotalItems, true

	case "completeItems":
		return &record.CompleteItems, true

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
