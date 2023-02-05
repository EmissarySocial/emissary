package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActivityStream struct {
	ActivityStreamID primitive.ObjectID `json:"activityStreamId" bson:"_id"`
	UserID           primitive.ObjectID `json:"userId"           bson:"userId"`
	ContentJSON      string             `json:"contentJson"      bson:"contentJson"`

	journal.Journal `json:"journal" bson:"journal"`
}

func NewActivityStream() ActivityStream {
	return ActivityStream{
		ActivityStreamID: primitive.NewObjectID(),
	}
}

func ActivityStreamSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"activityStreamId": schema.String{Format: "objectId"},
			"userId":           schema.String{Format: "objectId"},
			"contentJson":      schema.String{Format: "json"},
		},
	}
}

func (activityStream ActivityStream) ID() string {
	return activityStream.ActivityStreamID.Hex()
}
