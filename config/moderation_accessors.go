package config

import (
	"github.com/benpate/rosetta/schema"
)

// ModerationSchema returns the schema element for the Moderation config struct.
func ModerationSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"provider": schema.String{Enum: []string{ModerationProviderCoop}},
			"url":      schema.String{MaxLength: 255},
			"coop":     CoopSchema(),
		},
	}
}

// GetPointer resolves property paths for the Moderation config struct.
func (m *Moderation) GetPointer(name string) (any, bool) {

	switch name {

	case "provider":
		return &m.Provider, true

	case "url":
		return &m.URL, true

	case "coop":
		return &m.Coop, true
	}

	return nil, false
}
