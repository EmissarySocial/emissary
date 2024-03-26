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
			"groupIds":       id.SliceSchema(),
			"imageId":        schema.String{Format: "objectId"},
			"displayName":    schema.String{MaxLength: 64, Required: true},
			"statusMessage":  schema.String{MaxLength: 128},
			"location":       schema.String{MaxLength: 64},
			"links":          schema.Array{Items: PersonLinkSchema(), MaxLength: 6},
			"profileUrl":     schema.String{Format: "url"},
			"emailAddress":   schema.String{Format: "email", Required: true},
			"username":       schema.String{MaxLength: 32, Required: true},
			"locale":         schema.String{},
			"signupNote":     schema.String{MaxLength: 256},
			"inboxTemplate":  schema.String{MaxLength: 128},
			"outboxTemplate": schema.String{MaxLength: 128},
			"followerCount":  schema.Integer{},
			"followingCount": schema.Integer{},
			"ruleCount":      schema.Integer{},
			"isPublic":       schema.Boolean{},
			"isOwner":        schema.Boolean{},
			"data":           schema.Object{Wildcard: schema.String{}},
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

	case "links":
		return &user.Links, true

	case "isOwner":
		return &user.IsOwner, true

	case "isPublic":
		return &user.IsPublic, true

	case "followerCount":
		return &user.FollowerCount, true

	case "followingCount":
		return &user.FollowingCount, true

	case "ruleCount":
		return &user.RuleCount, true

	case "displayName":
		return &user.DisplayName, true

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

	default:
		return nil, false
	}
}

func (user *User) GetStringOK(name string) (string, bool) {
	switch name {

	case "userId":
		return user.UserID.Hex(), true

	case "imageId":
		return user.ImageID.Hex(), true

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

	case "imageId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			user.ImageID = objectID
			return true
		}

	}

	return false
}
