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
			"notes":           schema.String{MaxLength: 1024},
			"username":        schema.String{MaxLength: 128},
			"url":             schema.String{Required: true, MaxLength: 1024},
			"profileUrl":      schema.String{Format: "url", MaxLength: 1024},
			"iconUrl":         schema.String{Format: "url", MaxLength: 1024},
			"behavior":        schema.String{Enum: []string{FollowingBehaviorPosts, FollowingBehaviorPostsAndReplies}, Default: FollowingBehaviorPostsAndReplies, Required: true},
			"ruleAction":      schema.String{Enum: []string{FollowingRuleActionIgnore, RuleActionMute, RuleActionLabel, RuleActionBlock}, Default: RuleActionLabel, Required: true},
			"collapseThreads": schema.Boolean{Default: null.NewBool(true)},
			"isPublic":        schema.Boolean{Default: null.NewBool(false)},
			"method":          schema.String{Enum: []string{FollowingMethodPoll, FollowingMethodWebSub, FollowingMethodActivityPub}},
			"status":          schema.String{Enum: []string{FollowingStatusNew, FollowingStatusLoading, FollowingStatusSuccess, FollowingStatusFailure}},
			"statusMessage":   schema.String{MaxLength: 1024},
			"lastPolled":      schema.Integer{Minimum: null.NewInt64(0), BitSize: 64},
			"pollDuration":    schema.Integer{Minimum: null.NewInt64(1)},
			"purgeDuration":   schema.Integer{Minimum: null.NewInt64(0)},
			"nextPoll":        schema.Integer{Minimum: null.NewInt64(0), BitSize: 64},
			"errorCount":      schema.Integer{Minimum: null.NewInt64(0)},
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

	case "notes":
		return &following.Notes, true

	case "username":
		return &following.Username, true

	case "url":
		return &following.URL, true

	case "profileUrl":
		return &following.ProfileURL, true

	case "iconUrl":
		return &following.IconURL, true

	case "behavior":
		return &following.Behavior, true

	case "ruleAction":
		return &following.RuleAction, true

	case "collapseThreads":
		return &following.CollapseThreads, true

	case "isPublic":
		return &following.IsPublic, true

	case "method":
		return &following.Method, true

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
