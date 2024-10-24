package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func WebhookSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"webhookId": schema.String{Format: "objectID"},
			"label":     schema.String{},
			"targetUrl": schema.String{Format: "url"},
			"events": schema.Array{Items: schema.String{Enum: []string{
				WebhookEventStreamCreate,
				WebhookEventStreamUpdate,
				WebhookEventStreamDelete,
				WebhookEventUserCreate,
				WebhookEventUserUpdate,
				WebhookEventUserDelete,
				WebhookEventStreamPublish,
				WebhookEventStreamUnpublish,
			}}},
		},
	}
}

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

func (webhook Webhook) GetStringOK(name string) (string, bool) {

	switch name {

	case "webhookId":
		return webhook.WebhookID.Hex(), true
	}

	return "", false
}

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
