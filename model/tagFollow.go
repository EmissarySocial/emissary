package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TagFollow records that a User follows a #hashtag, so tagged posts reach their home timeline
type TagFollow struct {
	TagFollowID primitive.ObjectID `json:"tagFollowId" bson:"_id"`    // Unique ID of this TagFollow
	UserID      primitive.ObjectID `json:"userId"      bson:"userId"` // ID of the User who follows the tag
	Name        string             `json:"name"        bson:"name"`   // Tag as the User typed it, without the leading #
	Value       string             `json:"value"       bson:"value"`  // Normalized tag (ToToken), used for matching

	journal.Journal `json:"-" bson:",inline"`
}

// NewTagFollow returns a fully initialized TagFollow
func NewTagFollow() TagFollow {
	return TagFollow{
		TagFollowID: primitive.NewObjectID(),
	}
}

// ID returns the unique identifier for this TagFollow, implementing the data.Object interface
func (tagFollow TagFollow) ID() string {
	return tagFollow.TagFollowID.Hex()
}
