package model

import (
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserSummary is used as a lightweight, read-only summary of a user record.
type UserSummary struct {
	UserID        primitive.ObjectID `bson:"_id"`
	IconID        primitive.ObjectID `bson:"iconId"`
	DisplayName   string             `bson:"displayName"`
	EmailAddress  string             `bson:"emailAddress"`
	Username      string             `bson:"username"`
	ProfileURL    string             `bson:"profileUrl"`
	StatusMessage string             `bson:"statusMessage"`
	Hashtags      sliceof.String     `bson:"hashtags"`
	Shuffle       int64              `bson:"shuffle"`
	MovedTo       string             `bson:"movedTo"`
}

func NewUserSummary() UserSummary {
	return UserSummary{
		UserID:        primitive.NewObjectID(),
		IconID:        primitive.NilObjectID,
		DisplayName:   "",
		EmailAddress:  "",
		Username:      "",
		ProfileURL:    "",
		StatusMessage: "",
		Hashtags:      sliceof.NewString(),
	}
}

func UserSummaryFields() []string {
	return []string{"_id", "displayName", "emailAddress", "username", "iconId", "profileUrl", "statusMessage", "hashtags", "shuffle", "movedTo"}
}

func (userSummary UserSummary) Fields() []string {
	return UserSummaryFields()
}

func (userSummary UserSummary) IconURL() string {
	if userSummary.IconID.IsZero() {
		return ""
	}
	return "/@" + userSummary.UserID.Hex() + "/attachments/" + userSummary.IconID.Hex()
}
