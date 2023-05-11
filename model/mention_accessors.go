package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func MentionSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"mentionId": schema.String{Format: "objectId"},
			"objectId":  schema.String{Format: "objectId"},
			"type":      schema.String{Enum: []string{MentionTypeStream, MentionTypeUser}},
			"stateId":   schema.String{Enum: []string{MentionStatusValidated, MentionStatusPending, MentionStatusInvalid}},
			"origin":    OriginLinkSchema(),
			"author":    PersonLinkSchema(),
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

func (mention *Mention) GetPointer(name string) (any, bool) {

	switch name {

	case "origin":
		return &mention.Origin, true

	case "author":
		return &mention.Author, true

	case "type":
		return &mention.Type, true

	case "stateId":
		return &mention.StateID, true
	}

	return nil, false
}

func (mention *Mention) GetStringOK(name string) (string, bool) {
	switch name {

	case "mentionId":
		return mention.MentionID.Hex(), true

	case "objectId":
		return mention.ObjectID.Hex(), true

	}

	return "", false
}

func (mention *Mention) SetString(name string, value string) bool {
	switch name {

	case "mentionId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			mention.MentionID = objectID
			return true
		}

	case "objectId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			mention.ObjectID = objectID
			return true
		}
	}

	return false
}
