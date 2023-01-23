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
			"displayName":    schema.String{MaxLength: 64},
			"statusMessage":  schema.String{MaxLength: 128},
			"location":       schema.String{MaxLength: 64},
			"links":          schema.Array{Items: PersonLinkSchema(), MaxLength: 6},
			"profileUrl":     schema.String{Format: "url"},
			"emailAddress":   schema.String{Format: "email"},
			"username":       schema.String{MaxLength: 32, Required: true},
			"followerCount":  schema.Integer{},
			"followingCount": schema.Integer{},
			"blockCount":     schema.Integer{},
			"isOwner":        schema.Boolean{},
		},
	}
}

/*********************************
 * Getter Interfaces
 *********************************/

func (user *User) GetBoolOK(name string) (bool, bool) {
	switch name {

	case "isOwner":
		return user.IsOwner, true

	default:
		return false, false
	}
}

func (user *User) GetIntOK(name string) (int, bool) {
	switch name {

	case "followerCount":
		return user.FollowerCount, true

	case "followingCount":
		return user.FollowingCount, true

	case "blockCount":
		return user.BlockCount, true

	default:
		return 0, false
	}
}

func (user *User) GetStringOK(name string) (string, bool) {
	switch name {

	case "userId":
		return user.UserID.Hex(), true

	case "imageId":
		return user.ImageID.Hex(), true

	case "displayName":
		return user.DisplayName, true

	case "statusMessage":
		return user.StatusMessage, true

	case "location":
		return user.Location, true

	case "emailAddress":
		return user.EmailAddress, true

	case "username":
		return user.Username, true

	case "profileUrl":
		return user.ProfileURL, true

	default:
		return "", false
	}
}

/*********************************
 * Setter Interfaces
 *********************************/

func (user *User) SetBool(name string, value bool) bool {
	switch name {

	case "isOwner":
		user.IsOwner = value
		return true
	}
	return false
}

func (user *User) SetInt(name string, value int) bool {

	switch name {

	case "followerCount":
		user.FollowerCount = value
		return true

	case "followingCount":
		user.FollowingCount = value
		return true

	case "blockCount":
		user.BlockCount = value
		return true
	}

	return false
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

	case "displayName":
		user.DisplayName = value
		return true

	case "statusMessage":
		user.StatusMessage = value
		return true

	case "location":
		user.Location = value
		return true

	case "emailAddress":
		user.EmailAddress = value
		return true

	case "username":
		user.Username = value
		return true

	case "profileUrl":
		user.ProfileURL = value
		return true
	}

	return false
}

/*********************************
 * Tree Traversal
 *********************************/

func (user *User) GetObject(name string) (any, bool) {

	switch name {

	case "groupIds":
		return &user.GroupIDs, true

	case "links":
		return &user.Links, true

	default:
		return nil, false
	}
}
