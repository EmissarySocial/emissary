package model

import (
	"github.com/benpate/rosetta/maps"
)

type Client struct {
	ProviderID string   `bson:"provider" path:"providerId"` // ID of the provider that this credential accesses
	Data       maps.Map `bson:"data"     path:"data"`       // Unique data for this credential
	Active     bool     `bson:"active"   path:"active"`     // Is this credential active?
}

func NewClient(providerID string) Client {
	return Client{
		ProviderID: providerID,
		Data:       maps.New(),
	}
}

// ID implements the set.Value interface
func (client Client) ID() string {
	return client.ProviderID
}
