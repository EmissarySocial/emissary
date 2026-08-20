package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WebhookSchema returns the rosetta schema that describes a Webhook
func WebhookSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"webhookId": schema.String{Format: "objectId"},
			"label":     schema.String{Format: "text", MaxLength: 64},
			"targetUrl": schema.String{Format: "url"},
			"events": schema.Array{Items: schema.String{Enum: []string{
				WebhookEventStreamCreate,
				WebhookEventStreamUpdate,
				WebhookEventStreamDelete,
				WebhookEventUserCreate,
				WebhookEventUserUpdate,
				WebhookEventUserDelete,
				WebhookEventStreamPublish,
				WebhookEventStreamPublishUndo,
				WebhookEventStreamSyndicate,
				WebhookEventStreamSyndicateUndo,
			}}},
		},
	}
}

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (webhook *Webhook) GetPointer(name string) (any, bool) {

	switch name {

	case "events":
		return &webhook.Events, true

	case "label":
		return &webhook.Label, true

	case "targetUrl":
		return &webhook.TargetURL, true
	}

	return nil, false
}

// GetStringOK returns the named property. Implements schema.StringGetter.
func (webhook Webhook) GetStringOK(name string) (string, bool) {

	switch name {

	case "webhookId":
		return webhook.WebhookID.Hex(), true
	}

	return "", false
}

// SetString writes the named property. Implements schema.StringSetter.
func (webhook *Webhook) SetString(name string, value string) bool {

	switch name {

	case "webhookId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			webhook.WebhookID = objectID
			return true
		}
	}

	return false
}
