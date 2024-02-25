package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OutboxMessage represents a single item in a User's inbox or outbox.  It is loosely modelled on the OutboxMessageStreams
// standard, and can be converted into a strict go-fed streams.Type object.
type OutboxMessage struct {
	OutboxMessageID primitive.ObjectID `json:"messageId"  bson:"_id"`        // Unique ID of the OutboxMessage
	ParentType      string             `json:"parentType" bson:"parentType"` // Type of the parent object (User or Stream)
	ParentID        primitive.ObjectID `json:"parentId"   bson:"parentId"`   // Unique ID of the User who owns this OutboxMessage (in their inbox or outbox)
	ActivityType    string             `json:"type"       bson:"type"`       // Type of the activity (Create, Follow, Like, Block, etc.)
	URL             string             `json:"url"        bson:"url"`        // URL of the object (if applicable)

	journal.Journal `json:"-" bson:",inline"`
}

// NewOutboxMessage returns a fully initialized OutboxMessage record
func NewOutboxMessage() OutboxMessage {
	return OutboxMessage{
		OutboxMessageID: primitive.NewObjectID(),
	}
}

func OutboxMessageFields() []string {
	return []string{"url"}
}

func (summary OutboxMessage) Fields() []string {
	return OutboxMessageFields()
}

/******************************************
 * data.Object Interface
 ******************************************/

func (message OutboxMessage) ID() string {
	return message.OutboxMessageID.Hex()
}
