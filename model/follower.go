package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Follower struct {
	FollowerID primitive.ObjectID `path:"followerId" json:"followerId" bson:"_id"`    // Unique identifier for this Follower
	UserID     primitive.ObjectID `path:"userId"     json:"userId"     bson:"userId"` // Unique identifier for the User that is being followed
	Actor      PersonLink         `path:"actor"      json:"actor"      bson:"actor"`  // Person who is following the User
	Method     string             `path:"method"     json:"method"     bson:"method"` // Method of following (e.g. "RSS", "RSSCloud", "ActivityPub".)

	journal.Journal `path:"journal" json:"journal" bson:"journal"`
}

func NewFollower() Follower {
	return Follower{}
}

func FollowerSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"followerId": schema.String{Format: "objectId"},
			"userId":     schema.String{Format: "objectId"},
			"actor":      PersonLinkSchema(),
			"method":     schema.String{Enum: []string{FollowMethodRSS, FollowMethodRSSCloud, FollowMethodActivityPub}},
		},
	}
}

func (follower *Follower) ID() string {
	return follower.FollowerID.Hex()
}
