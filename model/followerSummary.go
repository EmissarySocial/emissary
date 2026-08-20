package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FollowerSummary is an abbreviated Follower, used when listing many Followers at once
type FollowerSummary struct {
	FollowerID primitive.ObjectID `bson:"_id"`      // Unique identifier for this Follower
	ParentID   primitive.ObjectID `bson:"parentId"` // Unique identifier for the User that is being followed
	Actor      PersonLink         `bson:"actor"`    // Person who is follower the User
	Method     string             `bson:"method"`   // Method of follower (e.g. "POLL", "EMAIL", "ActivityPub".)
}

// FollowerSummaryFields returns a slice of all BSON field names for a FollowerSummary
func FollowerSummaryFields() []string {
	return []string{"_id", "parentId", "actor", "method"}
}

// Fields returns the database fields required to populate a FollowerSummary
func (summary FollowerSummary) Fields() []string {
	return FollowerSummaryFields()
}

/******************************************
 * Other Methods
 ******************************************/

// Icon returns the name of the icon that represents this Follower's delivery method
func (summary FollowerSummary) Icon() string {
	switch summary.Method {

	case FollowerMethodEmail:
		return "email"
	case FollowerMethodActivityPub:
		return "activitypub"
	}

	return ""
}
