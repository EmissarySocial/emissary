package model

import (
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OutboxMessageSummary is an abbreviated OutboxMessage, used when paging through an outbox
type OutboxMessageSummary struct {
	OutboxMessageID primitive.ObjectID `bson:"_id"`
	ObjectID        string             `bson:"objectId"`
	CreateDate      int64              `bson:"createDate"` // Unix epoch MILLISECONDS (journal projection; used as an opaque paging cursor)
}

// NewOutboxMessageSummary returns a fully initialized, empty OutboxMessageSummary
func NewOutboxMessageSummary() OutboxMessageSummary {
	return OutboxMessageSummary{}
}

// OutboxMessageSummaryFields returns the database fields required to populate a OutboxMessageSummary
func OutboxMessageSummaryFields() []string {
	return []string{"objectId", "createDate"}
}

// Created returns the creation date of this OutboxMessageSummary, as a Unix timestamp
func (message OutboxMessageSummary) Created() int64 {
	return message.CreateDate
}

// ID returns the primary key of this OutboxMessageSummary, as a string
func (message OutboxMessageSummary) ID() string {
	return message.OutboxMessageID.Hex()
}

// ActivityPubURL returns the ActivityPub URL of this OutboxMessageSummary
func (message OutboxMessageSummary) ActivityPubURL() string {
	return message.ObjectID
}

// GetJSONLD returns this OutboxMessageSummary as a JSON-LD document
func (message OutboxMessageSummary) GetJSONLD() mapof.Any {
	return mapof.Any{
		vocab.PropertyID: message.ObjectID,
	}
}
