package model

import (
	"crypto/subtle"
	"net/url"
	"slices"
	"strings"

	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OAuthClient is a third-party application that has been registered to use this server's OAuth API
type OAuthClient struct {
	ClientID     primitive.ObjectID `json:"clientId"     bson:"_id"`          // Unique identifier for this Client record
	ClientSecret string             `json:"-"            bson:"clientSecret"` // Shared secret used to retrieve OAuth Tokens
	ClientURL    string             `json:"clientUrl"    bson:"clientUrl"`    // CIMD URL of the actor that created this Client
	Name         string             `json:"name"         bson:"name"`         // Human-friendly name of the Client
	Summary      string             `json:"summary"      bson:"summary"`      // Human-friendly summary/description of the Client
	IconURL      string             `json:"iconUrl"      bson:"iconUrl"`      // URL of an icon image to display with the Client's name
	Website      string             `json:"website"      bson:"website"`      // Human-friendly website URL for the Client
	RedirectURIs sliceof.String     `json:"redirectUris" bson:"redirectUris"` // Slice of URLs that the Client is allowed to redirect Users to
	Scopes       sliceof.String     `json:"scopes"       bson:"scopes"`       // OAuth authorization scopes approved for use by this Client
	MastodonApp  bool               `json:"mastodonApp"  bson:"mastodonApp"`  // TRUE if a Mastodon app registered this Client through POST /api/v1/apps (set by the server only)

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

// AllowsRedirectURI returns TRUE if this client may send a User to the provided redirect_uri: one it lists,
// or, for a URL client that lists none, any URL on the client's own origin (IndieAuth §4.2.2)
func (app OAuthClient) AllowsRedirectURI(redirectURI string) bool {

	// RULE: There is no redirect to an empty address
	if redirectURI == "" {
		return false
	}

	// A listed redirect_uri always matches exactly
	if slices.Contains(app.RedirectURIs, redirectURI) {
		return true
	}

	// RULE: Only a URL client may use its own origin...
	if app.ClientURL == "" {
		return false
	}

	// RULE: ...and only when it lists nothing, because a published list is the client's own, stricter rule
	if len(app.RedirectURIs) > 0 {
		return false
	}

	return isSameOrigin(app.ClientURL, redirectURI)
}

// isSameOrigin returns TRUE if a redirect_uri is an http(s) URL on the same scheme, host, and port as a client_id
func isSameOrigin(clientID string, redirectURI string) bool {

	// RULE: A redirect_uri carries no fragment and no credentials (RFC 6749 §3.1.2)
	if strings.Contains(redirectURI, "#") {
		return false
	}

	client, err := url.Parse(clientID)

	if err != nil {
		return false
	}

	redirect, err := url.Parse(redirectURI)

	if err != nil {
		return false
	}

	if redirect.User != nil {
		return false
	}

	// Compare scheme, host, and port, filling in the port each scheme implies
	if !strings.EqualFold(client.Scheme, redirect.Scheme) {
		return false
	}

	if (client.Hostname() == "") || !strings.EqualFold(client.Hostname(), redirect.Hostname()) {
		return false
	}

	return effectivePort(client) == effectivePort(redirect)
}

// effectivePort returns a URL's port, or the default port for its scheme when it names none
func effectivePort(value *url.URL) string {

	if port := value.Port(); port != "" {
		return port
	}

	switch strings.ToLower(value.Scheme) {

	case "http":
		return "80"

	case "https":
		return "443"
	}

	return ""
}

// Toot converts this object into a Mastodon-compatible Application object
func (app OAuthClient) Toot() object.Application {
	return object.Application{
		ID:           app.ClientID.Hex(),
		Name:         app.Name,
		Website:      app.Website,
		Scopes:       app.Scopes,
		RedirectURIs: app.RedirectURIs,
		RedirectURI:  strings.Join(app.RedirectURIs, "\n"),
		ClientID:     first.String(app.ClientURL, app.ClientID.Hex()),
		ClientSecret: app.ClientSecret,
	}
}
