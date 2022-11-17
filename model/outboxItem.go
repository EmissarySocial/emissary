package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OutboxItem struct {
	OutboxItemID    primitive.ObjectID `json:"inboxItemId" bson:"_id"`         // Unique ID for this message
	UserID          primitive.ObjectID `json:"userId"      bson:"userId"`      // ID of the User who received this message
	StreamID        primitive.ObjectID `json:"streamId"    bson:"streamId"`    // ID of the Stream that created this message
	ReplyTo         ReplyToLink        `json:"replyTo"     bson:"replyTo"`     // Link to the message that this OutboxItem is a reply to
	Role            string             `json:"role"        bson:"role"`        // Social role that this outbox item performs
	Label           string             `json:"label"       bson:"label"`       // Label/Name/Subject of the message
	Summary         string             `json:"summary"     bson:"summary"`     // Short summar of the message
	ContentHTML     string             `json:"content"     bson:"content"`     // HTML content of the message
	PublishDate     int64              `json:"publishDate" bson:"publishDate"` // Date/Time that this message was published
	journal.Journal `json:"-" bson:"journal"`
}

func NewOutboxItem() OutboxItem {
	return OutboxItem{
		OutboxItemID: primitive.NewObjectID(),
	}
}

func OutboxItemSchema() schema.Element {
	return schema.Object{}
}

func (item OutboxItem) ID() string {
	return item.OutboxItemID.Hex()
}

func (item OutboxItem) GetObjectID(name string) (primitive.ObjectID, error) {
	switch name {
	case "userId":
		return item.UserID, nil
	case "streamId":
		return item.StreamID, nil
	case "replyTo.internalId":
		return item.ReplyTo.InternalID, nil
	default:
		return primitive.NilObjectID, derp.NewBadRequestError("model.OutboxItem", "GetObjectID", name, "Invalid field name")
	}
}

func (item OutboxItem) GetString(name string) (string, error) {
	switch name {
	case "role":
		return item.Role, nil
	case "label":
		return item.Label, nil
	case "summary":
		return item.Summary, nil
	case "contentHTML":
		return item.ContentHTML, nil
	case "replyTo.label":
		return item.ReplyTo.Label, nil
	case "replyTo.url":
		return item.ReplyTo.URL, nil
	default:
		return "", derp.NewBadRequestError("model.OutboxItem", "GetString", name, "Invalid field name")
	}
}

func (item OutboxItem) GetInt64(name string) (int64, error) {
	switch name {
	case "publishDate":
		return item.PublishDate, nil
	case "createDate":
		return item.Journal.CreateDate, nil
	case "updateDate":
		return item.Journal.UpdateDate, nil
	case "replyTo.updateDate":
		return item.ReplyTo.UpdateDate, nil
	default:
		return 0, derp.NewBadRequestError("model.OutboxItem", "GetInt64", name, "Invalid field name")
	}
}
