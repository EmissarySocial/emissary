package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OutboxItem struct {
	OutboxItemID    primitive.ObjectID `json:"inboxItemId" bson:"_id"`      // Unique ID for this message
	UserID          primitive.ObjectID `json:"userId"      bson:"userId"`   // ID of the User who received this message
	StreamID        primitive.ObjectID `json:"streamId"    bson:"streamId"` // ID of the Stream that created this message
	ReplyTo         ReplyToLink        `json:"replyTo"     bson:"replyTo"`  // Link to the message that this OutboxItem is a reply to
	Label           string             `json:"label"       bson:"label"`    // Label/Name/Subject of the message
	Summary         string             `json:"summary"     bson:"summary"`  // Short summar of the message
	ContentHTML     string             `json:"content"     bson:"content"`  // HTML content of the message
	journal.Journal `json:"-" bson:"journal"`
}

func NewOutboxItem() OutboxItem {
	return OutboxItem{
		OutboxItemID: primitive.NewObjectID(),
	}
}

func (item OutboxItem) ID() string {
	return item.OutboxItemID.Hex()
}
