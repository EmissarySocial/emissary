package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
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
	return Follower{
		FollowerID: primitive.NewObjectID(),
	}
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
	case "userId":
		return follower.UserID, nil
	}

	return primitive.NilObjectID, derp.NewInternalError("model.follower.GetObjectID", "Invalid property", name)
}

func (follower *Follower) GetString(name string) (string, error) {
	switch name {
	case "method":
		return follower.Method, nil
	}

	return "", derp.NewInternalError("model.follower.GetString", "Invalid property", name)
}

func (follower *Follower) GetInt(name string) (int, error) {
	return 0, derp.NewInternalError("model.follower.GetInt", "Invalid property", name)
}

func (follower *Follower) GetInt64(name string) (int64, error) {
	return 0, derp.NewInternalError("model.follower.GetInt64", "Invalid property", name)
}

func (follower *Follower) GetBool(name string) (bool, error) {
	return false, derp.NewInternalError("model.follower.GetBool", "Invalid property", name)
}
