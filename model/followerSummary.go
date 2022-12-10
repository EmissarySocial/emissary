package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FollowerSummary struct {
	FollowerID primitive.ObjectID `path:"followerId" json:"followerId" bson:"_id"`    // Unique identifier for this Follower
	UserID     primitive.ObjectID `path:"userId"     json:"userId"     bson:"userId"` // Unique identifier for the User that is being followed
	Actor      PersonLink         `path:"actor"      json:"actor"      bson:"actor"`  // Person who is follower the User
	Method     string             `path:"method"     json:"method"     bson:"method"` // Method of follower (e.g. "RSS", "RSSCloud", "ActivityPub".)
}

// FollowerSummaryFields returns a slice of all BSON field names for a FollowerSummary
func FollowerSummaryFields() []string {
	return []string{"_id", "userId", "actor", "method"}
}

func (followerSummary FollowerSummary) Fields() []string {
	return FollowerSummaryFields()
}
