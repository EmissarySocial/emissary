package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Webhook defines an outbound webhook that can be triggered by events in the system
type Webhook struct {
	WebhookID       primitive.ObjectID `json:"webhookId" bson:"_id"`
	Events          sliceof.String     `json:"events"    bson:"events"`
	Label           string             `json:"label"     bson:"label"`
	TargetURL       string             `json:"targetUrl" bson:"targetUrl"`
	journal.Journal `json:"-" bson:",inline"`
}

// NewWebhook returns a fully initialized Webhook object
func NewWebhook() Webhook {
	return Webhook{
		WebhookID: primitive.NewObjectID(),
		Events:    sliceof.NewString(),
	}
}

func WebhookFields() []string {
	return []string{"_id", "events", "label", "targetUrl"}
}

func (userSummary Webhook) Fields() []string {
	return WebhookFields()
}

// ID returns the unique identifier for this Webhook, and is required to implement the data.Object interface
func (webhook Webhook) ID() string {
	return webhook.WebhookID.Hex()
}
