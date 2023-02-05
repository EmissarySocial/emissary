package model

import (
	"net/url"
	"strconv"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActivityStream struct {
	ActivityStreamID primitive.ObjectID      `json:"activityStreamId" bson:"_id"`
	URI              string                  `json:"uri"              bson:"uri"`
	UserID           primitive.ObjectID      `json:"userId"           bson:"userId"`
	Container        ActivityStreamContainer `json:"container"        bson:"container"`
	PublishDate      int64                   `json:"publishDate"      bson:"publishDate"`
	Content          mapof.Any               `json:"content"          bson:"content"`

	journal.Journal `json:"journal" bson:"journal"`
}

func NewActivityStream(container ActivityStreamContainer) ActivityStream {
	return ActivityStream{
		ActivityStreamID: primitive.NewObjectID(),
		Container:        container,
		Content:          mapof.NewAny(),
	}
}

func NewInboxActivityStream() ActivityStream {
	return NewActivityStream(ActivityStreamContainerInbox)
}

func NewOutboxActivityStream() ActivityStream {
	return NewActivityStream(ActivityStreamContainerOutbox)
}

func (activityStream ActivityStream) ID() string {
	return activityStream.ActivityStreamID.Hex()
}

func (activityStream ActivityStream) PublishDateString() string {
	return strconv.FormatInt(activityStream.PublishDate, 10)
}

func (activityStream ActivityStream) URL() *url.URL {
	result, _ := url.Parse(activityStream.URI)
	return result
}

func (activityStream *ActivityStream) UpdateWithActivityStream(other *ActivityStream) {
	activityStream.PublishDate = other.PublishDate
	activityStream.Content = other.Content
}
