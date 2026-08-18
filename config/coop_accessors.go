package config

import (
	"github.com/benpate/rosetta/schema"
)

// CoopSchema returns the schema element for the Coop config struct.
func CoopSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"apiKey":           schema.String{MaxLength: 255},
			"webhookPublicKey": schema.String{MaxLength: 4096},
			"userItemTypeId":   schema.String{MaxLength: 255},
			"statusItemTypeId": schema.String{MaxLength: 255},
			"suspendActionId":  schema.String{MaxLength: 255},
			"silenceActionId":  schema.String{MaxLength: 255},
			"deleteActionId":   schema.String{MaxLength: 255},
		},
	}
}

// GetPointer resolves property paths for the Coop config struct.
func (c *Coop) GetPointer(name string) (any, bool) {

	switch name {

	case "apiKey":
		return &c.APIKey, true

	case "webhookPublicKey":
		return &c.WebhookPublicKey, true

	case "userItemTypeId":
		return &c.UserItemTypeID, true

	case "statusItemTypeId":
		return &c.StatusItemTypeID, true

	case "suspendActionId":
		return &c.SuspendActionID, true

	case "silenceActionId":
		return &c.SilenceActionID, true

	case "deleteActionId":
		return &c.DeleteActionID, true
	}

	return nil, false
}
