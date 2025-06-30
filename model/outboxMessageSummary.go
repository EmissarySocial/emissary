package model

import (
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OutboxMessageSummary struct {
	OutboxMessageID primitive.ObjectID `bson:"_id"`
	ObjectID        string             `bson:"objectId"`
	CreateDate      int64              `bson:"createDate"`
}

func NewOutboxMessageSummary() OutboxMessageSummary {
	return OutboxMessageSummary{}
}

func OutboxMessageSummaryFields() []string {
	return []string{"objectId", "createDate"}
}

func (message OutboxMessageSummary) Created() int64 {
	return message.CreateDate
}

func (message OutboxMessageSummary) ID() string {
	return message.OutboxMessageID.Hex()
}

func (message OutboxMessageSummary) ActivityPubURL() string {
	return message.ObjectID
}

func (message OutboxMessageSummary) GetJSONLD() mapof.Any {
	return mapof.Any{
		vocab.PropertyID: message.ObjectID,
	}
}
