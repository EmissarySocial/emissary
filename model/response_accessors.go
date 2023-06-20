package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ResponseSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"responseId": schema.String{Format: "objectId"},
			"userId":     schema.String{Format: "objectId"},
			"actorId":    schema.String{Format: "url"},
			"objectId":   schema.String{Format: "url"},
			"type":       schema.String{MaxLength: 128},
			"summary":    schema.String{MaxLength: 256},
			"content":    schema.String{MaxLength: 256},
		},
	}
}

func (response *Response) GetPointer(name string) (any, bool) {

	switch name {

	case "actorId":
		return &response.ActorID, true

	case "objectId":
		return &response.ObjectID, true

	case "type":
		return &response.Type, true

	case "summary":
		return &response.Summary, true

	case "content":
		return &response.Content, true
	}

	return nil, false
}

func (response Response) GetStringOK(name string) (string, bool) {
	switch name {

	case "responseId":
		return response.ResponseID.Hex(), true

	case "userId":
		return response.UserID.Hex(), true
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

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			response.UserID = objectID
			return true
		}
	}

	return false
}
