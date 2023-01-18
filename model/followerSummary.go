package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FollowerSummary struct {
	FollowerID primitive.ObjectID `json:"summaryId" bson:"_id"`      // Unique identifier for this Follower
	ParentID   primitive.ObjectID `json:"parentId"  bson:"parentId"` // Unique identifier for the User that is being followed
	Actor      PersonLink         `json:"actor"     bson:"actor"`    // Person who is follower the User
	Method     string             `json:"method"    bson:"method"`   // Method of follower (e.g. "RSS", "RSSCloud", "ActivityPub".)
}

// FollowerSummaryFields returns a slice of all BSON field names for a FollowerSummary
func FollowerSummaryFields() []string {
	return []string{"_id", "parentId", "actor", "method"}
}

func (summary FollowerSummary) Fields() []string {
	return FollowerSummaryFields()
}

/*******************************************
 * Other Methods
 *******************************************/

func (summary FollowerSummary) MethodIcon() string {
	switch summary.Method {
	case FollowMethodPoll:
		return "rss-fill"
	case FollowMethodWebSub:
		return "websub-fill"
	case FollowMethodActivityPub:
		return "activitypub-fill"
	}

	return ""
}
