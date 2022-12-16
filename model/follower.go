package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
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
			"method":     schema.String{Enum: []string{FollowMethodPoll, FollowMethodWebSub, FollowMethodRSSCloud, FollowMethodActivityPub}},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

func (follower *Follower) ID() string {
	return follower.FollowerID.Hex()
}

func (follower *Follower) GetObjectID(name string) (primitive.ObjectID, error) {
	switch name {
	case "followerId":
		return follower.FollowerID, nil
	case "parentId":
		return follower.ParentID, nil
	}

	return primitive.NilObjectID, derp.NewInternalError("model.follower.GetObjectID", "Invalid property", name)
}

func (follower *Follower) GetString(name string) (string, error) {
	switch name {
	case "type":
		return follower.Type, nil
	case "method":
		return follower.Method, nil
	}

	return "", derp.NewInternalError("model.follower.GetString", "Invalid property", name)
}

func (follower *Follower) GetInt(name string) (int, error) {
	return 0, derp.NewInternalError("model.follower.GetInt", "Invalid property", name)
}

func (follower *Follower) GetInt64(name string) (int64, error) {
	switch name {
	case "expireDate":
		return follower.ExpireDate, nil
	}
	return 0, derp.NewInternalError("model.follower.GetInt64", "Invalid property", name)
}

func (follower *Follower) GetBool(name string) (bool, error) {
	return false, derp.NewInternalError("model.follower.GetBool", "Invalid property", name)
}
