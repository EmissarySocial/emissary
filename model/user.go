package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a person or machine account that can own pages and sections.
type User struct {
	UserID       primitive.ObjectID `json:"userId"       bson:"_id"`
	RoomID       primitive.ObjectID `json:"roomId"       bson:"roomId"`
	DisplayName  string             `json:"displayName"  bson:"displayName"`
	EmailAddress string             `json:"emailAddress" bson:"emailAddress"`
	MobilePhone  string             `json:"mobilePhone"  bson:"mobilePhone"`

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the primary key for this record
func (user *User) ID() string {
	return user.UserID.Hex()
}
