package model

import (
	"github.com/benpate/rosetta/maps"
	"golang.org/x/oauth2"
)

// Client represents a single connection to an individual Provider.  It usually contains an OAuth2 token, but may also contain
// other connection information like a username or password.  It may also represent a connection that is still being formed,
// for instance, storing the intermediate state of an OAuth2 connection that has not yet completed the three-legged handshake.
type Client struct {
	ProviderID string        `bson:"provider" path:"providerId"` // ID of the provider that this credential accesses
	Data       maps.Map      `bson:"data"     path:"data"`       // Unique data for this credential
	Token      *oauth2.Token `bson:"token"    path:"token"`      // OAuth2 Token (if necessary)
	Active     bool          `bson:"active"   path:"active"`     // Is this credential active?
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

/**************************************
 * Data Accessor Methods
 **************************************/

// GetString is a shortcut to the Data.GetString() method
func (client Client) GetString(key string) string {
	return client.Data.GetString(key)
}

// SetString is a shortcut to the Data.SetString() method
func (client *Client) SetString(key string, value string) {
	client.Data.SetString(key, value)
}

// GetInt is a shortcut to the Data.GetInt() method
func (client Client) GetInt(key string) int {
	return client.Data.GetInt(key)
}

// SetInt is a shortcut to the Data.SetInt() method
func (client *Client) SetInt(key string, value int) {
	client.Data.SetInt(key, value)
}

// GetInt64 is a shortcut to the Data.GetInt64() method
func (client Client) GetInt64(key string) int64 {
	return client.Data.GetInt64(key)
}

// SetInt64 is a shortcut to the Data.SetInt64() method
func (client *Client) SetInt64(key string, value int64) {
	client.Data.SetInt64(key, value)
}
