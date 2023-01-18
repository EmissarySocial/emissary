package model

import "go.mongodb.org/mongo-driver/bson/primitive"

/******************************************
 * Schema Getter/Setter Interfaces
 ******************************************/

func (following Following) GetIntOK(name string) (int, bool) {

	switch name {

	case "pollDuration":
		return following.PollDuration, true

	case "purgeDuration":
		return following.PurgeDuration, true

	case "errorCount":
		return following.ErrorCount, true

	}

	return 0, false
}

func (following Following) GetInt64OK(name string) (int64, bool) {

	switch name {

	case "lastPolled":
		return following.LastPolled, true

	case "nextPoll":
		return following.NextPoll, true

	}

	return 0, false
}

func (following Following) GetStringOK(name string) (string, bool) {

	switch name {

	case "followingId":
		return following.FollowingID.Hex(), true

	case "userId":
		return following.UserID.Hex(), true

	case "folderId":
		return following.FolderID.Hex(), true

	case "label":
		return following.Label, true

	case "url":
		return following.URL, true

	case "method":
		return following.Method, true

	case "status":
		return following.Status, true

	case "statusMessage":
		return following.StatusMessage, true

	}

	return "", false
}

func (following *Following) SetIntOK(name string, value int) bool {

	switch name {

	case "pollDuration":
		following.PollDuration = value
		return true

	case "purgeDuration":
		following.PurgeDuration = value
		return true

	case "errorCount":
		following.ErrorCount = value
		return true
	}

	return false
}

func (following *Following) SetInt64OK(name string, value int64) bool {

	switch name {

	case "lastPolled":
		following.LastPolled = value
		return true

	case "nextPoll":
		following.NextPoll = value
		return true
	}

	return false
}

func (following *Following) SetStringOK(name string, value string) bool {

	switch name {

	case "followingId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			following.FollowingID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			following.UserID = objectID
			return true
		}

	case "folderId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			following.FolderID = objectID
			return true
		}

	case "label":
		following.Label = value
		return true

	case "url":
		following.URL = value
		return true

	case "method":
		following.Method = value
		return true

	case "status":
		following.Status = value
		return true

	case "statusMessage":
		following.StatusMessage = value
		return true
	}

	return false
}
