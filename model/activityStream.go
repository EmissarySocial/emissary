package model

import (
	"strconv"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActivityStream struct {
	ActivityStreamID primitive.ObjectID      `json:"activityStreamId" bson:"_id"`
	UserID           primitive.ObjectID      `json:"userId"           bson:"userId"`
	Container        ActivityStreamContainer `json:"container"        bson:"container"`
	PublishDate      int64                   `json:"publishDate"      bson:"publishDate"`
	Content          map[string]any          `json:"content"          bson:"content"`

	journal.Journal `json:"journal" bson:"journal"`
}

func NewActivityStream(container ActivityStreamContainer) ActivityStream {
	return ActivityStream{
		ActivityStreamID: primitive.NewObjectID(),
		Container:        container,
	}
}

func NewInboxActivityStream() ActivityStream {
	return NewActivityStream(ActivityStreamContainerInbox)
}

func NewOutboxActivityStream() ActivityStream {
	return NewActivityStream(ActivityStreamContainerOutbox)
}

func ActivityStreamSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"activityStreamId": schema.String{Format: "objectId"},
			"userId":           schema.String{Format: "objectId"},
			"publishDate":      schema.Integer{BitSize: 64},
			"container":        schema.Integer{},
		},
	}
}

func (activityStream ActivityStream) ID() string {
	return activityStream.ActivityStreamID.Hex()
}

func (activityStream ActivityStream) PublishDateString() string {
	return strconv.FormatInt(activityStream.PublishDate, 10)
}
