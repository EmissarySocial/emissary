package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func FollowerSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"followerId": schema.String{Format: "objectId"},
			"parentId":   schema.String{Format: "objectId"},
			"type":       schema.String{Enum: []string{FollowerTypeSearch, FollowerTypeStream, FollowerTypeUser}},
			"method":     schema.String{Enum: []string{FollowerMethodActivityPub, FollowerMethodEmail, FollowerMethodWebSub}},
			"format":     schema.String{Enum: []string{MimeTypeActivityPub, MimeTypeAtom, MimeTypeHTML, MimeTypeJSONFeed, MimeTypeRSS, MimeTypeXML}},
			"stateId":    schema.String{Enum: []string{FollowerStateActive, FollowerStatePending}},
			"actor":      PersonLinkSchema(),
			"data":       schema.Object{Wildcard: schema.String{MaxLength: 256}},
			"expireDate": schema.Integer{BitSize: 64},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (follower *Follower) GetPointer(name string) (any, bool) {

	switch name {

	case "actor":
		return &follower.Actor, true

	case "data":
		return &follower.Data, true

	case "expireDate":
		return &follower.ExpireDate, true

	case "type":
		return &follower.ParentType, true

	case "method":
		return &follower.Method, true

	case "format":
		return &follower.Format, true

	case "stateId":
		return &follower.StateID, true
	}

	return nil, false
}

func (follower *Follower) GetStringOK(name string) (string, bool) {
	switch name {

	case "followerId":
		return follower.FollowerID.Hex(), true

	case "parentId":
		return follower.ParentID.Hex(), true
	}

	return "", false
}

func (follower *Follower) SetString(name string, value string) bool {

	switch name {

	case "followerId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			follower.FollowerID = objectID
			return true
		}

	case "parentId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			follower.ParentID = objectID
			return true
		}
	}

	return false
}

/******************************************
 * Tree Traversal
 ******************************************/
