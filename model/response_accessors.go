package model

import (
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ResponseSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"responseId": schema.String{Format: "objectId"},
			"userId":     schema.String{Format: "objectId"},
			"actor":      schema.String{Format: "url"},
			"object":     schema.String{Format: "url"},
			"type":       schema.String{MaxLength: 128, Enum: []string{vocab.ActivityTypeAnnounce, vocab.ActivityTypeLike, vocab.ActivityTypeDislike}},
			"content":    schema.String{MaxLength: 256},
		},
	}
}

func (response *Response) GetPointer(name string) (any, bool) {

	switch name {

	case "actor":
		return &response.Actor, true

	case "object":
		return &response.Object, true

	case "type":
		return &response.Type, true

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
