package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// UserSummary is used as a lightweight, read-only summary of a user record.
type UserSummary struct {
	UserID       primitive.ObjectID `bson:"_id"`
	DisplayName  string             `bson:"displayName"`
	EmailAddress string             `bson:"emailAddress"`
	Username     string             `bson:"username"`
	ImageURL     string             `bson:"imageUrl"`
	ProfileURL   string             `bson:"profileUrl"`
}

func NewUserSummary() UserSummary {
	return UserSummary{
		UserID:       primitive.NewObjectID(),
		DisplayName:  "",
		EmailAddress: "",
		Username:     "",
		ImageURL:     "",
		ProfileURL:   "",
	}
}

func UserSummaryFields() []string {
	return []string{"_id", "displayName", "emailAddress", "username", "imageUrl", "profileUrl"}
}

func (userSummary UserSummary) Fields() []string {
	return UserSummaryFields()
}
