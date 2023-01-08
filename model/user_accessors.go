package model

import "go.mongodb.org/mongo-driver/bson/primitive"

/*********************************
 * Getter Methods
 *********************************/

func (user *User) GetBool(name string) bool {
	switch name {
	case "isOwner":
		return user.IsOwner
	default:
		return false
	}
}

func (user *User) GetInt(name string) int {
	switch name {
	case "followerCount":
		return user.FollowerCount
	case "followingCount":
		return user.FollowingCount
	case "blockCount":
		return user.BlockCount
	default:
		return 0
	}
}

func (user *User) GetString(name string) string {
	switch name {
	case "userId":
		return user.UserID.Hex()
	case "imageId":
		return user.ImageID.Hex()
	case "displayName":
		return user.DisplayName
	case "statusMessage":
		return user.StatusMessage
	case "location":
		return user.Location
	case "emailAddress":
		return user.EmailAddress
	case "username":
		return user.Username
	case "profileUrl":
		return user.ProfileURL
	default:
		return ""
	}
}

/*********************************
 * Setter Methods
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

func (user *User) GetChild(name string) (any, bool) {
	switch name {
	case "passwordReset":
		return &user.PasswordReset, true
	case "links":
		return &user.Links, true
	default:
		return nil, false
	}
}
