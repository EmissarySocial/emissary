package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OriginLinkSchema returns a JSON Schema for OriginLink structures
func OriginLinkSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"type":        schema.String{Enum: []string{OriginTypePrimary, OriginTypeLike, OriginTypeDislike, OriginTypeReply, OriginTypeAnnounce}},
			"followingId": schema.String{Format: "objectId"},
			"label":       schema.String{MaxLength: 128},
			"url":         schema.String{Format: "url"},
			"imageUrl":    schema.String{Format: "url"},
		},
	}
}

/*********************************
 * Getter Interfaces
 *********************************/

func (origin *OriginLink) GetPointer(name string) (any, bool) {
	switch name {

	case "type":
		return &origin.Type, true

	case "label":
		return &origin.Label, true

	case "url":
		return &origin.URL, true

	case "imageUrl":
		return &origin.ImageURL, true

	}

	return nil, false
}

func (origin *OriginLink) GetStringOK(name string) (string, bool) {
	switch name {

	case "followingId":
		return origin.FollowingID.Hex(), true
	}

	return "", false
}

/*********************************
 * Setter Interfaces
 *********************************/

func (origin *OriginLink) SetString(name string, value string) bool {
	switch name {

	case "followingId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			origin.FollowingID = objectID
			return true
		}
	}

	return false
}
