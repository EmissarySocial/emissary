package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const FollowerTypeStream = "Stream"

const FollowerTypeUser = "User"

type Follower struct {
	FollowerID      primitive.ObjectID `path:"followerId" json:"followerId" bson:"_id"`        // Unique identifier for this Follower
	ParentID        primitive.ObjectID `path:"parentId"   json:"parentId"   bson:"parentId"`   // Unique identifier for the Stream that is being followed (including user's outboxes)
	Type            string             `path:"type"       json:"type"       bson:"type"`       // Type of record being followed (e.g. "User", "Stream")
	Method          string             `path:"method"     json:"method"     bson:"method"`     // Method of following (e.g. "POLL", "WEBSUB", "RSS-CLOUD", "ACTIVITYPUB")
	Format          string             `path:"format"     json:"format"     bson:"format"`     // Format of the data being followed (e.g. "JSON", "XML", "ATOM", "RSS")
	Actor           PersonLink         `path:"actor"      json:"actor"      bson:"actor"`      // Person who is following the User
	Data            maps.Map           `path:"data"       json:"data"       bson:"data"`       // Additional data about this Follower that depends on the follow method
	ExpireDate      int64              `path:"expireDate" json:"expireDate" bson:"expireDate"` // Unix timestamp (in seconds) when this follower will be automatically purged.
	journal.Journal `path:"journal" json:"journal" bson:"journal"`
}

func NewFollower() Follower {
	return Follower{
		FollowerID: primitive.NewObjectID(),
		Data:       make(maps.Map),
	}
}

func FollowerSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"followerId": schema.String{Format: "objectId"},
			"parentId":   schema.String{Format: "objectId"},
			"type":       schema.String{Enum: []string{FollowerTypeStream, FollowerTypeUser}},
			"actor":      PersonLinkSchema(),
			"method":     schema.String{Enum: []string{FollowMethodPoll, FollowMethodWebSub, FollowMethodActivityPub}},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

func (follower *Follower) ID() string {
	return follower.FollowerID.Hex()
}
