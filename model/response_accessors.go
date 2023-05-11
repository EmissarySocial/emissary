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
			"origin":     OriginLinkSchema(),
			"object":     DocumentLinkSchema(),
			"objectId":   schema.String{Format: "objectId"},
			"type":       schema.String{MaxLength: 128},
			"value":      schema.String{MaxLength: 256},
		},
	}
}

func (response Response) GetStringOK(name string) (string, bool) {
	switch name {

	case "responseId":
		return response.ResponseID.Hex(), true

	case "objectId":
		return response.ObjectID.Hex(), true

	case "type":
		return response.Type, true

	case "value":
		return response.Value, true
	}

	return "", false
}

func (response *Response) GetPointer(name string) (any, bool) {

	switch name {

	case "actor":
		return &response.Actor, true

	case "origin":
		return &response.Origin, true

	case "object":
		return &response.Object, true
	}

	return nil, false
}

func (response *Response) SetString(name string, value string) bool {
	switch name {

	case "responseId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			response.ResponseID = objectID
			return true
		}

	case "objectId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			response.ObjectID = objectID
			return true
		}

	case "type":
		response.Type = value
		return true

	case "value":
		response.Value = value
		return true
	}

	return false
}
