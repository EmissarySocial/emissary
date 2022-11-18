package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"html/template"
)

type InboxItem struct {
	InboxItemID    primitive.ObjectID `json:"inboxItemId"   bson:"_id"`             // Unique ID for this message
	InboxFolderID  primitive.ObjectID `json:"inboxFolderId" bson:"inboxFolderId"`   // ID of the Inbox folder the contains this message
	SubscriptionID primitive.ObjectID `json:"subscriptionId" bson:"subscriptionId"` // ID of the subscription that generated this message
	UserID         primitive.ObjectID `json:"userId"        bson:"userId"`          // ID of the User who received this message
	Author         PersonLink         `json:"author"        bson:"author"`          // Link to the Author of this InboxItem
	Origin         OriginLink         `json:"origin"        bson:"origin"`          // Link to the origin of this InboxItem
	ReplyTo        ReplyToLink        `json:"replyTo"       bson:"replyTo"`         // Link to the message that this InboxItem is a reply to
	Label          string             `json:"label"         bson:"label"`           // Label/Name/Subject of the message
	Summary        string             `json:"summary"       bson:"summary"`         // Short summar of the message
	Content        string             `json:"content"       bson:"content"`         // HTML content of the message
	ImageURL       string             `json:"imageUrl"      bson:"imageUrl"`        // URL of an image associated with this message
	PublishDate    int64              `json:"publishDate"   bson:"publishDate"`     // Unix timestamp of the date/time when this message was published
	ReadDate       int64              `json:"readDate"      bson:"readDate"`        // Date when this message was read by the user

	journal.Journal `json:"-" bson:"journal"`
}

func NewInboxItem() InboxItem {
	return InboxItem{
		InboxItemID: primitive.NewObjectID(),
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

func (item *InboxItem) ID() string {
	return item.InboxItemID.Hex()
}

/*******************************************
 * schema.DataObject Interface
 *******************************************/

func (inboxItem *InboxItem) GetObjectID(name string) (primitive.ObjectID, error) {
	switch name {
	case "inboxItemId":
		return inboxItem.InboxItemID, nil
	case "inboxFolderId":
		return inboxItem.InboxFolderID, nil
	case "subscriptionId":
		return inboxItem.SubscriptionID, nil
	case "userId":
		return inboxItem.UserID, nil
	default:
		return primitive.NilObjectID, derp.NewInternalError("model.InboxItem.GetObjectID", "Unknown ObjectID", name)
	}
}

func (inboxItem *InboxItem) GetString(name string) (string, error) {
	switch name {
	case "label":
		return inboxItem.Label, nil
	case "summary":
		return inboxItem.Summary, nil
	case "content":
		return inboxItem.Content, nil
	case "imageUrl":
		return inboxItem.ImageURL, nil
	default:
		return "", derp.NewInternalError("model.InboxItem.GetString", "Unknown String", name)
	}
}

func (inboxItem *InboxItem) GetInt(name string) (int, error) {
	return 0, derp.NewInternalError("model.InboxItem.GetInt", "Unknown Int", name)
}
func (inboxItem *InboxItem) GetInt64(name string) (int64, error) {
	switch name {
	case "publishDate":
		return inboxItem.PublishDate, nil
	case "readDate":
		return inboxItem.ReadDate, nil
	default:
		return 0, derp.NewInternalError("model.InboxItem.GetInt64", "Unknown Int64", name)
	}
}

func (item InboxItem) Status() string {
	if item.ReadDate == 0 {
		return "unread"
	} else {
		return "read"
	}
}

func (item InboxItem) IsUnread() bool {
	return item.ReadDate == 0
}

func (item InboxItem) ContentHTML() template.HTML {
	return template.HTML(item.Content)
}

func (item InboxItem) IsIncomplete() bool {
	return true
}

func InboxItemSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"inboxItemId":    schema.String{Format: "objectId", Required: true},
			"inboxFolderId":  schema.String{Format: "objectId"},
			"subscriptionId": schema.String{Format: "objectId"},
			"userId":         schema.String{Format: "objectId", Required: true},
			"label":          schema.String{Required: true},
			"summary":        schema.String{},
			"content":        schema.String{Format: "html"},
		},
	}
}
