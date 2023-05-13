package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ResponseSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"responseId": schema.String{Format: "objectId"},
			"actor":      PersonLinkSchema(),
			"message":    DocumentLinkSchema(),
			"type":       schema.String{MaxLength: 128},
			"value":      schema.String{MaxLength: 256},
		},
	}
}

func (response *Response) GetPointer(name string) (any, bool) {

	switch name {

	case "actor":
		return &response.Actor, true

	case "message":
		return &response.Message, true

	case "type":
		return &response.Type, true

	case "value":
		return &response.Value, true
	}

	return nil, false
}

func (response Response) GetStringOK(name string) (string, bool) {
	switch name {

	case "responseId":
		return response.ResponseID.Hex(), true
	}

	return "", false
}

func (response *Response) SetString(name string, value string) bool {
	switch name {

	case "responseId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			response.ResponseID = objectID
			return true
		}
	}

	return false
}
