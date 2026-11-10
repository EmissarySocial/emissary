package build

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/form"
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
	return builder._factory.Domain().Cached().Summary()
}

// User returns a summary of the Authenticated User
func (builder OAuthAuthorization) User() model.UserSummary {
	return builder._user.Summary()
}

// ClientID returns the unique ID of the OAuth client being authorized
func (builder OAuthAuthorization) ClientID() string {
	return builder._client.ClientID.Hex()
}

// ClientURL returns the URL that a client identifies itself by (its client_id), or "" for a registered client
func (builder OAuthAuthorization) ClientURL() string {
	return builder._client.ClientURL
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

// IsOutOfBand returns TRUE if the result is shown on the page for the User to copy, instead of redirecting
func (builder OAuthAuthorization) IsOutOfBand() bool {
	return builder._transaction.RedirectURI == model.OAuthRedirectURIOutOfBand
}

// ResponseType returns the OAuth response type ("code" or "token") being requested
func (builder OAuthAuthorization) ResponseType() string {
	return builder._transaction.ResponseType
}

// Scope returns the raw, space-delimited scope string being requested
func (builder OAuthAuthorization) Scope() string {
	return builder._transaction.Scope
}

// Scopes returns each requested scope in plain words, grouped as the consent page lists them.
// An empty list requests nothing but the User's identity.
func (builder OAuthAuthorization) Scopes() []form.LookupCode {
	return describeScopes(builder._transaction.Scopes())
}

// State returns the opaque state value that is echoed back to the OAuth client
func (builder OAuthAuthorization) State() string {
	return builder._transaction.State
}

// CodeChallenge returns the PKCE challenge (RFC 7636) that the client sent, if any
func (builder OAuthAuthorization) CodeChallenge() string {
	return builder._transaction.CodeChallenge
}

// CodeChallengeMethod returns the PKCE challenge method ("plain" or "S256") that the client sent, if any
func (builder OAuthAuthorization) CodeChallengeMethod() string {
	return builder._transaction.CodeChallengeMethod
}

// Me returns the identity URL that the client suggested, which is an unverified hint (IndieAuth §5.2)
func (builder OAuthAuthorization) Me() string {
	return builder._transaction.Me
}

// SignedInAsSomeoneElse returns TRUE if the client's `me` hint names a User of this domain other than the signed-in one
func (builder OAuthAuthorization) SignedInAsSomeoneElse() bool {
	return signedInAsSomeoneElse(builder._transaction.Me, builder.Hostname(), builder._user)
}

// SwitchAccountURL returns the sign-out URL that brings the visitor back to this request once they sign in again
func (builder OAuthAuthorization) SwitchAccountURL() string {

	if builder._request == nil {
		return "/signout"
	}

	return "/signout?next=" + url.QueryEscape(builder._request.URL.RequestURI())
}

// describeScopes returns each requested scope once, in plain words: the dataset's scopes first, in its order,
// then any other scope by name, in the order requested
func describeScopes(requested []string) []form.LookupCode {

	// Collect each distinct scope.  The grant ignores empty scopes, and so does the page.
	remaining := make(map[string]bool, len(requested))

	for _, value := range requested {
		if value != "" {
			remaining[value] = true
		}
	}

	result := make([]form.LookupCode, 0, len(remaining))

	// Describe the scopes the dataset knows, in its order, so that each group is one run
	for _, lookupCode := range dataset.OAuthScopes() {
		if remaining[lookupCode.Value] {
			result = append(result, lookupCode)
			delete(remaining, lookupCode.Value)
		}
	}

	// Name every other scope, so that the User sees everything that is being asked for
	for _, value := range requested {
		if remaining[value] {
			result = append(result, form.LookupCode{Value: value, Label: value, Description: "A permission this server does not recognize.", Icon: "key", Group: "Other"})
			delete(remaining, value)
		}
	}

	return result
}

// signedInAsSomeoneElse returns TRUE if a `me` hint is the profile URL, on this host, of someone other than the
// signed-in User.  The domain root, another host, and any other URL name nobody to switch to.
func signedInAsSomeoneElse(me string, hostname string, user *model.User) bool {

	// RULE: With nobody signed in, there is nobody to compare
	if user == nil {
		return false
	}

	parsed, err := url.Parse(me)

	if err != nil {
		return false
	}

	// RULE: Only a URL on this host can name one of its Users
	if parsed.Host == "" {
		return false
	}

	if !strings.EqualFold(parsed.Hostname(), hostname) {
		return false
	}

	// RULE: Only a profile path, `/@name`, names a User
	name, isProfilePath := strings.CutPrefix(strings.TrimSuffix(parsed.Path, "/"), "/@")

	if !isProfilePath {
		return false
	}

	if name == "" {
		return false
	}

	if strings.Contains(name, "/") {
		return false
	}

	// The signed-in User answers to "me", their ID, and their username, which matches without regard to case
	if name == "me" {
		return false
	}

	if name == user.UserID.Hex() {
		return false
	}

	return !strings.EqualFold(name, user.Username)
}
