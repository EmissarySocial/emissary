package build

import (
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/uri"
)

// OAuthAuthorization is a lightweight builder that
// displays UI pages for an OAuth Application.
type OAuthAuthorization struct {
	_service *service.OAuthClient
	_client  model.OAuthClient

	// _transaction is the OAuth request being authorized.  It is NOT the HTTP request,
	// which the embedded Common already carries as `_request`.
	_transaction model.OAuthAuthorizationRequest
	_user        *model.User

	Theme
}

// NewOAuthAuthorization returns a fully initialized/loaded `OAuthAuthorization` builder
func NewOAuthAuthorization(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, transaction model.OAuthAuthorizationRequest, user *model.User) (OAuthAuthorization, error) {

	const location = "build.NewOAuthAuthorization"

	// Create the result object.  Theme is embedded because the consent page is rendered by
	// a handler rather than by a Template action, and Theme carries the accessors -- and
	// the "noindex" rule -- that every such page needs.
	result := OAuthAuthorization{
		_service:     factory.OAuthClient(),
		_client:      model.NewOAuthClient(),
		_transaction: transaction,
		_user:        user,

		Theme: NewTheme(factory, session, request, response),
	}

	// Try to load the OAuthClient object
	if err := result._service.LoadOrCreateByClientToken(session, transaction.ClientID, &result._client); err != nil {
		return OAuthAuthorization{}, derp.Wrap(err, location, "Loading OAuth Application")
	}

	// Validate the transaction
	if err := result._transaction.Validate(result._client); err != nil {
		return OAuthAuthorization{}, derp.Wrap(err, location, "Invalid authorization request")
	}

	// Return success.
	return result, nil
}

// Domain returns a summary of the current Domain
func (builder OAuthAuthorization) Domain() model.DomainSummary {
	return builder._factory.Domain().Get().Summary()
}

// User returns a summary of the Authenticated User
func (builder OAuthAuthorization) User() model.UserSummary {
	return builder._user.Summary()
}

// ClientID returns the unique ID of the OAuth client being authorized
func (builder OAuthAuthorization) ClientID() string {
	return builder._client.ClientID.Hex()
}

// Name returns the human-friendly name of the OAuth client
func (builder OAuthAuthorization) Name() string {
	return builder._client.Name
}

// IconURL returns the icon image that represents the OAuth client
func (builder OAuthAuthorization) IconURL() string {
	return builder._client.IconURL
}

// Website returns the public website of the OAuth client
func (builder OAuthAuthorization) Website() string {
	if website := builder._client.Website; website != "" {
		return uri.PrependProtocol(website)
	}

	if clientURL := uri.Hostname(builder._client.ClientURL); clientURL != "" {
		return uri.PrependProtocol(clientURL)
	}

	return ""
}

// RedirectURI returns the callback URI where the user is sent after authorizing
func (builder OAuthAuthorization) RedirectURI() string {
	return builder._transaction.RedirectURI
}

// ResponseType returns the OAuth response type ("code" or "token") being requested
func (builder OAuthAuthorization) ResponseType() string {
	return builder._transaction.ResponseType
}

// Scope returns the raw, space-delimited scope string being requested
func (builder OAuthAuthorization) Scope() string {
	return builder._transaction.Scope
}

// Scopes returns the individual scopes being requested, defaulting to "read"
func (builder OAuthAuthorization) Scopes() []string {

	if builder._transaction.Scope == "" {
		return []string{"read"}
	}

	return strings.Split(builder._transaction.Scope, " ")
}

// State returns the opaque state value that is echoed back to the OAuth client
func (builder OAuthAuthorization) State() string {
	return builder._transaction.State
}
