package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func StreamResponseSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"streamResponseId": schema.String{Format: "objectId", Required: true},
			"stream":           DocumentLinkSchema(),
			"actor":            PersonLinkSchema(),
			"origin":           OriginLinkSchema(),
			"type":             schema.String{Enum: []string{ResponseTypeLike, ResponseTypeDislike, ResponseTypeMention, ResponseTypeRepost}, Required: true},
			"value":            schema.String{MaxLength: 64},
		},
	}
}

func (response *StreamResponse) GetPointer(name string) (any, bool) {

	switch name {

	case "stream":
		return &response.Stream, true

	case "actor":
		return &response.Actor, true

	case "origin":
		return &response.Origin, true

	case "type":
		return &response.Type, true

	case "value":
		return &response.Value, true
	}

	return nil, false
}

func (response *StreamResponse) GetStringOK(name string) (string, bool) {

	switch name {

	case "streamResponseId":
		return response.StreamResponseID.Hex(), true
	}

	return "", false
}

func (response *StreamResponse) SetString(name string, value string) bool {

	switch name {

	case "streamResponseId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			response.StreamResponseID = objectID
			return true
		}
	}

	return false
}
