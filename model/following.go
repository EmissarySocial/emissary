package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const FollowMethodRSS = "RSS"

const FollowMethodRSSCloud = "RSSCloud"

const FollowMethodActivityPub = "ActivityPub"

type Following struct {
	FollowingID primitive.ObjectID `path:"followingId" json:"followingId" bson:"_id"`    // Unique identifier for this Following
	UserID      primitive.ObjectID `path:"userId"      json:"userId"      bson:"userId"` // Unique identifier for the User that is being followed
	Object      PersonLink         `path:"object"      json:"object"      bson:"object"` // Person who is following the User
	Method      string             `path:"method"      json:"method"      bson:"method"` // Method of following (e.g. "RSS", "RSSCloud", "ActivityPub".)

	journal.Journal `path:"journal" json:"journal" bson:"journal"`
}

func NewFollowing() Following {
	return Following{}
}

func FollowingSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"followingId": schema.String{Format: "objectId"},
			"userId":      schema.String{Format: "objectId"},
			"actor":       PersonLinkSchema(),
			"method":      schema.String{Enum: []string{FollowMethodRSS, FollowMethodRSSCloud, FollowMethodActivityPub}},
		},
	}
}

func (following *Following) ID() string {
	return following.FollowingID.Hex()
}
