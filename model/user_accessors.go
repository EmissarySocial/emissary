package model

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

func (user *User) GetChild(name string) any {
	switch name {
	case "passwordReset":
		return &user.PasswordReset
	default:
		return nil
	}
}
