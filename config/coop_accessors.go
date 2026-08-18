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
	}

	return nil, false
}
