package model

import (
	"github.com/benpate/rosetta/maps"
	"golang.org/x/oauth2"
)

// Client represents a single connection to an individual Provider.  It usually contains an OAuth2 token, but may also contain
// other connection information like a username or password.  It may also represent a connection that is still being formed,
// for instance, storing the intermediate state of an OAuth2 connection that has not yet completed the three-legged handshake.
type Client struct {
	ProviderID string        `bson:"provider"` // ID of the provider that this credential accesses
	Data       maps.Map      `bson:"data"`     // Unique data for this credential
	Token      *oauth2.Token `bson:"token"`    // OAuth2 Token (if necessary)
	Active     bool          `bson:"active"`   // Is this credential active?
}

// NewClient returns a fully initialized Client object.
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
