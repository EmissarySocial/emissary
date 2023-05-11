package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OutboxMessage represents a single item in a User's inbox or outbox.  It is loosely modelled on the OutboxMessageStreams
// standard, and can be converted into a strict go-fed streams.Type object.
type OutboxMessage struct {
	OutboxMessageID primitive.ObjectID `json:"messageId"   bson:"_id"`        // Unique ID of the OutboxMessage
	ObjectType      string             `json:"objectType"  bson:"objectType"` // Type internal record that generated this OutboxMessage (Stream, Block, Follow, Reaction, etc)
	ObjectID        primitive.ObjectID `json:"objectId"    bson:"objectId"`   // Unique ID of the internal record that generated this OutboxMessage
	UserID          primitive.ObjectID `json:"userId"      bson:"userId"`     // Unique ID of the User who owns this OutboxMessage (in their inbox or outbox)
	ParentID        primitive.ObjectID `json:"parentId"    bson:"parentId"`   // Unique ID of the parent object (if applicable)
	Activity        mapof.Any          `json:"activity"    bson:"activity"`   // ActivityPub Document that is the subject of this OutboxMessage
	Rank            int64              `json:"rank"        bson:"rank"`       // Sort rank for this message (publishDate * 1000 + sequence number)

	journal.Journal `json:"-" bson:"journal"`
}

// NewOutboxMessage returns a fully initialized OutboxMessage record
func NewOutboxMessage() OutboxMessage {
	return OutboxMessage{
		OutboxMessageID: primitive.NewObjectID(),
		Activity:        mapof.NewAny(),
	}
}

func OutboxMessageFields() []string {
	return []string{"activity", "rank"}
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

/******************************************
 * RoleStateEnumerator Methods
 ******************************************/

// State returns the current state of this Stream.  It is
// part of the implementation of the RoleStateEmulator interface
func (message OutboxMessage) State() string {
	return ""
}

// Roles returns a list of all roles that match the provided authorization
func (message OutboxMessage) Roles(authorization *Authorization) []string {
	return []string{MagicRoleMyself}
}

/******************************************
 * Other Methods
 ******************************************/

func (message OutboxMessage) RankSeconds() int64 {
	return message.Rank / 1000
}

func (message OutboxMessage) GetJSONLD() mapof.Any {
	return message.Activity
}
