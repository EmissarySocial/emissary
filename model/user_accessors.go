package model

import (
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func UserSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"userId":         schema.String{Format: "objectId"},
			"mapIds":         schema.Object{Wildcard: schema.String{}},
			"groupIds":       id.SliceSchema(),
			"iconId":         schema.String{Format: "objectId"},
			"imageId":        schema.String{Format: "objectId"},
			"iconUrl":        schema.String{Format: "url"}, // This is my first attempt at a "virtual field"
			"imageUrl":       schema.String{Format: "url"}, // This is my first attempt at a "virtual field"
			"displayName":    schema.String{MaxLength: 64, Required: true},
			"statusMessage":  schema.String{MaxLength: 1024},
			"location":       schema.String{MaxLength: 64},
			"links":          schema.Array{Items: PersonLinkSchema(), MaxLength: 6},
			"profileUrl":     schema.String{Format: "url"},
			"emailAddress":   schema.String{Format: "email", Required: true},
			"username":       schema.String{MaxLength: 32, Required: true},
			"locale":         schema.String{},
			"signupNote":     schema.String{MaxLength: 256},
			"stateId":        schema.String{},
			"inboxTemplate":  schema.String{MaxLength: 128},
			"outboxTemplate": schema.String{MaxLength: 128},
			"followerCount":  schema.Integer{},
			"followingCount": schema.Integer{},
			"ruleCount":      schema.Integer{},
			"isPublic":       schema.Boolean{},
			"isOwner":        schema.Boolean{},
			"isIndexable":    schema.Boolean{},
			"data":           schema.Object{Wildcard: schema.String{}},
			"hashtags":       schema.Array{Items: schema.String{Format: "token"}},
		},
	}
}

/*********************************
 * Getter/Setter Interfaces
 *********************************/

func (user *User) GetPointer(name string) (any, bool) {

	switch name {

	case "groupIds":
		return &user.GroupIDs, true

	case "mapIds":
		return &user.MapIDs, true

	case "links":
		return &user.Links, true

	case "isOwner":
		return &user.IsOwner, true

	case "isPublic":
		return &user.IsPublic, true

	case "isIndexable":
		return &user.IsIndexable, true

	case "followerCount":
		return &user.FollowerCount, true

	case "followingCount":
		return &user.FollowingCount, true

	case "ruleCount":
		return &user.RuleCount, true

	case "displayName":
		return &user.DisplayName, true

	case "stateId":
		return &user.StateID, true

	case "statusMessage":
		return &user.StatusMessage, true

	case "location":
		return &user.Location, true

	case "emailAddress":
		return &user.EmailAddress, true

	case "username":
		return &user.Username, true

	case "locale":
		return &user.Locale, true

	case "signupNote":
		return &user.SignupNote, true

	case "profileUrl":
		return &user.ProfileURL, true

	case "inboxTemplate":
		return &user.InboxTemplate, true

	case "outboxTemplate":
		return &user.OutboxTemplate, true

	case "data":
		return &user.Data, true

	case "hashtags":
		return &user.Hashtags, true

	default:
		return nil, false
	}
}

func (user *User) GetStringOK(name string) (string, bool) {
	switch name {

	case "userId":
		return user.UserID.Hex(), true

	case "iconId":
		return user.IconID.Hex(), true

	case "imageId":
		return user.ImageID.Hex(), true

	case "iconUrl":
		return user.ActivityPubIconURL(), true

	case "imageUrl":
		return user.ActivityPubImageURL(), true

	default:
		return "", false
	}
}

func (user *User) SetString(name string, value string) bool {

	switch name {

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			user.UserID = objectID
			return true
		}

	case "iconId":

		if value == "" {
			user.IconID = primitive.NilObjectID
			return true
		}

		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			user.IconID = objectID
			return true
		}

	case "imageId":

		if value == "" {
			user.ImageID = primitive.NilObjectID
			return true
		}

		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			user.ImageID = objectID
			return true
		}

	case "iconUrl":
		return true // Fail silently, but do not set iconUrl from this string

	case "imageUrl":
		return true // Fail silently, but do not set imageUrl from this string

	}

	return false
}
