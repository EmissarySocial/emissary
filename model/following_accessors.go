package model

import (
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FollowingSchema returns a validating schema for Following objects
func FollowingSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"followingId":     schema.String{Format: "objectId"},
			"userId":          schema.String{Format: "objectId"},
			"folderId":        schema.String{Format: "objectId", Required: true},
			"label":           schema.String{MaxLength: 128},
			"url":             schema.String{Required: true, MaxLength: 1024},
			"profileUrl":      schema.String{Format: "url", MaxLength: 1024},
			"imageUrl":        schema.String{Format: "url", MaxLength: 1024},
			"collapseThreads": schema.Boolean{},
			"method":          schema.String{Enum: []string{FollowMethodPoll, FollowMethodWebSub, FollowMethodActivityPub}},
			"format":          schema.String{Enum: []string{FollowingFormatActivityStream, FollowingFormatRSS, FollowingFormatAtom, FollowingFormatJSONFeed, FollowingFormatMicroFormats}},
			"status":          schema.String{Enum: []string{FollowingStatusNew, FollowingStatusLoading, FollowingStatusSuccess, FollowingStatusFailure}},
			"statusMessage":   schema.String{MaxLength: 1024},
			"lastPolled":      schema.Integer{Minimum: null.NewInt64(0), BitSize: 64},
			"pollDuration":    schema.Integer{Minimum: null.NewInt64(1)},
			"purgeDuration":   schema.Integer{Minimum: null.NewInt64(0)},
			"nextPoll":        schema.Integer{Minimum: null.NewInt64(0), BitSize: 64},
			"errorCount":      schema.Integer{Minimum: null.NewInt64(0)},
			"doMoveMessages":  schema.Boolean{},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (following *Following) GetPointer(name string) (any, bool) {
	switch name {

	case "label":
		return &following.Label, true

	case "url":
		return &following.URL, true

	case "profileUrl":
		return &following.ProfileURL, true

	case "imageUrl":
		return &following.ImageURL, true

	case "collapseThreads":
		return &following.CollapseThreads, true

	case "method":
		return &following.Method, true

	case "format":
		return &following.Format, true

	case "secret":
		// Do not allow access to "secret" field
		// return &following.Secret, true
		return nil, false

	case "status":
		return &following.Status, true

	case "statusMessage":
		return &following.StatusMessage, true

	case "lastPolled":
		return &following.LastPolled, true

	case "pollDuration":
		return &following.PollDuration, true

	case "purgeDuration":
		return &following.PurgeDuration, true

	case "nextPoll":
		return &following.NextPoll, true

	case "errorCount":
		return &following.ErrorCount, true

	case "doMoveMessages":
		return &following.DoMoveMessages, true
	}
	return nil, false
}

func (following Following) GetStringOK(name string) (string, bool) {

	switch name {

	case "followingId":
		return following.FollowingID.Hex(), true

	case "userId":
		return following.UserID.Hex(), true

	case "folderId":
		return following.FolderID.Hex(), true
	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

func (following *Following) SetBool(name string, value bool) bool {

	switch name {
	case "doMoveMessages":
		following.DoMoveMessages = value
		return true
	}

	return false
}

func (following *Following) SetInt(name string, value int) bool {

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

func (following *Following) SetString(name string, value string) bool {

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
	}

	return false
}
