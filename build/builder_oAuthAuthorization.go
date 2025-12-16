package build

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
)

// OAuthAuthorization is a lightweight builder that
// displays UI pages for an OAuth Application.
type OAuthAuthorization struct {
	_service       *service.OAuthClient
	_domainService *service.Domain
	_client        model.OAuthClient
	_request       model.OAuthAuthorizationRequest
	_user          *model.User
}

// NewOAuthAuthorization returns a fully initialized/loaded `OAuthAuthorization` builder
func NewOAuthAuthorization(factory Factory, session data.Session, request model.OAuthAuthorizationRequest, user *model.User) (OAuthAuthorization, error) {

	const location = "build.NewOAuthAuthorization"

	// Create the result object
	result := OAuthAuthorization{
		_service:       factory.OAuthClient(),
		_domainService: factory.Domain(),
		_client:        model.NewOAuthClient(),
		_request:       request,
		_user:          user,
	}

	// Try to load the OAuthClient object
	if err := result._service.LoadOrCreateByClientToken(session, request.ClientID, &result._client); err != nil {
		return OAuthAuthorization{}, derp.Wrap(err, location, "Unable to load OAuth Application")
	}

	// Validate the transaction
	if err := result._request.Validate(result._client); err != nil {
		return OAuthAuthorization{}, derp.Wrap(err, location, "Invalid authorization request")
	}

	// Return success.
	return result, nil
}

// Domain returns a summary of the current Domain
func (builder OAuthAuthorization) Domain() model.DomainSummary {
	return builder._domainService.Get().Summary()
}

// User returns a summary of the Authenticated User
func (builder OAuthAuthorization) User() model.UserSummary {
	return builder._user.Summary()
}

func (builder OAuthAuthorization) ClientID() string {
	return builder._client.ClientID.Hex()
}

func (builder OAuthAuthorization) Name() string {
	return builder._client.Name
}

func (builder OAuthAuthorization) IconURL() string {
	return builder._client.IconURL
}

func (builder OAuthAuthorization) Website() string {
	if website := builder._client.Website; website != "" {
		return dt.AddProtocol(website)
	}

	if clientURL := dt.NameOnly(builder._client.ClientURL); clientURL != "" {
		return dt.AddProtocol(clientURL)
	}

	return ""
}

func (builder OAuthAuthorization) RedirectURI() string {
	return builder._request.RedirectURI
}

func (builder OAuthAuthorization) ResponseType() string {
	return builder._request.ResponseType
}

func (builder OAuthAuthorization) Scope() string {
	return builder._request.Scope
}

func (builder OAuthAuthorization) Scopes() []string {

	if builder._request.Scope == "" {
		return []string{"read"}
	}

	return strings.Split(builder._request.Scope, " ")
}

func (builder OAuthAuthorization) State() string {
	return builder._request.State
}
