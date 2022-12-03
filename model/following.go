package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
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
	return Following{
		FollowingID: primitive.NewObjectID(),
	}
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

/*******************************************
 * data.Object Interface
 *******************************************/

func (following *Following) ID() string {
	return following.FollowingID.Hex()
}

func (following *Following) GetObjectID(name string) (primitive.ObjectID, error) {
	switch name {
	case "followingId":
		return following.FollowingID, nil
	case "userId":
		return following.UserID, nil
	}

	return primitive.NilObjectID, derp.NewInternalError("model.following.GetObjectID", "Invalid property", name)
}

func (following *Following) GetString(name string) (string, error) {
	switch name {
	case "method":
		return following.Method, nil
	}

	return "", derp.NewInternalError("model.following.GetString", "Invalid property", name)
}

func (following *Following) GetInt(name string) (int, error) {
	return 0, derp.NewInternalError("model.following.GetInt", "Invalid property", name)
}

func (following *Following) GetInt64(name string) (int64, error) {
	return 0, derp.NewInternalError("model.following.GetInt64", "Invalid property", name)
}

func (following *Following) GetBool(name string) (bool, error) {
	return false, derp.NewInternalError("model.following.GetBool", "Invalid property", name)
}
