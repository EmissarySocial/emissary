package model

import (
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

type OutboxMessageSummary struct {
	URL        string `bson:"url"`
	CreateDate int64  `bson:"createDate"`
}

func NewOutboxMessageSummary() OutboxMessageSummary {
	return OutboxMessageSummary{}
}

func OutboxMessageSummaryFields() []string {
	return []string{"url", "createDate"}
}

func (message OutboxMessageSummary) Created() int64 {
	return message.CreateDate
}

func (message OutboxMessageSummary) GetJSONLD() mapof.Any {
	return mapof.Any{
		vocab.PropertyID: message.URL,
	}
}
