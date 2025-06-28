package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OutboxMessage represents a single item in a User's inbox or outbox.  It is loosely modelled on the OutboxMessageStreams
// standard, and can be converted into a strict go-fed streams.Type object.
type OutboxMessage struct {
	OutboxMessageID primitive.ObjectID `json:"messageId"     bson:"_id"`           // Unique ID of the OutboxMessage
	ActorID         primitive.ObjectID `json:"actorId"       bson:"actorId"`       // Unique ID of the User who owns this OutboxMessage (in their inbox or outbox)
	ActorType       string             `json:"actorType"     bson:"actorType"`     // Type of the parent object (User or Stream)
	ActivityType    string             `json:"activityType"  bson:"activityType"`  // Type of the activity (Create, Follow, Like, Block, etc.)
	ObjectID        string             `json:"objectId"      bson:"objectId"`      // URL of the object (if applicable)
	PublishedDate   int64              `json:"publishedDate" bson:"publishedDate"` // Date when this OutboxMessage was published
	Permissions     Permissions        `json:"permissions"   bson:"permissions"`   // List of permissions for this OutboxMessage

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
 * JSONLDGetter Interface
 ******************************************/

func (message OutboxMessage) Created() int64 {
	return message.Journal.CreateDate
}

/******************************************
 * data.Object Interface
 ******************************************/

func (message OutboxMessage) ID() string {
	return message.OutboxMessageID.Hex()
}
