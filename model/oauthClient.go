package model

import (
	"crypto/subtle"

	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OAuthClient is a third-party application that has been registered to use this server's OAuth API
type OAuthClient struct {
	ClientID     primitive.ObjectID `bson:"_id"`          // Unique identifier for this Client record
	ClientSecret string             `bson:"clientSecret"` // Shared secret used to retrieve OAuth Tokens
	ClientURL    string             `bson:"clientUrl"`    // CIMD URL of the actor that created this Client
	Name         string             `bson:"name"`         // Human-friendly name of the Client
	Summary      string             `bson:"summary"`      // Human-friendly summary/description of the Client
	IconURL      string             `bson:"iconUrl"`      // URL of an icon image to display with the Client's name
	Website      string             `bson:"website"`      // Human-friendly website URL for the Client
	RedirectURIs sliceof.String     `bson:"redirectUris"` // Slice of URLs that the Client is allowed to redirect Users to
	Scopes       sliceof.String     `bson:"scopes"`       // OAuth authorization scopes approved for use by this Client

	journal.Journal `json:"-" bson:",inline"`
}

// NewOAuthClient returns a fully initialized, empty OAuthClient
func NewOAuthClient() OAuthClient {
	return OAuthClient{
		ClientID:     primitive.NewObjectID(),
		RedirectURIs: make([]string, 0),
		Scopes:       make([]string, 0),
	}
}

// ID returns the primary key of this OAuthClient, as a string
func (app OAuthClient) ID() string {
	return app.ClientID.Hex()
}

// IsConfidential reports whether this client authenticates with a shared secret.
// Public clients -- native/mobile apps and clients registered via a Client ID
// Metadata Document (FEP-d8c2) -- have no secret and MUST instead prove
// possession of their authorization code via PKCE (RFC 8252 / OAuth 2.1).
func (app OAuthClient) IsConfidential() bool {
	return app.ClientSecret != ""
}

// ValidateSecret confirms a supplied client_secret against this client's stored
// secret.  It is only meaningful for confidential clients; a public client has
// no secret and must be authenticated via PKCE instead.
func (app OAuthClient) ValidateSecret(clientSecret string) error {

	const location = "model.OAuthClient.ValidateSecret"

	// RULE: An empty secret is never a valid credential.  This also prevents a
	// public client's empty stored secret from being satisfied by an empty
	// supplied secret ("" == "").
	if clientSecret == "" {
		return derp.BadRequest(location, "Invalid client_secret")
	}

	// RULE: Constant-time comparison avoids leaking the secret through timing.
	if subtle.ConstantTimeCompare([]byte(app.ClientSecret), []byte(clientSecret)) != 1 {
		return derp.BadRequest(location, "Invalid client_secret")
	}

	return nil
}

// ToToot converts this object into a Mastodon-compatible Application object
func (app OAuthClient) Toot() object.Application {
	return object.Application{
		Name:         app.Name,
		Website:      app.Website,
		ClientID:     first.String(app.ClientURL, app.ClientID.Hex()),
		ClientSecret: app.ClientSecret,
	}
}
