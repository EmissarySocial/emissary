package render

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
)

// OAuthAuthorization renderer displays UI pages for an OAuth Application
type OAuthAuthorization struct {
	_service *service.OAuthApplication
	_app     model.OAuthApplication
	_request model.OAuthAuthorizationRequest
}

// NewOAuthAuthorization returns a fully initialized/loaded OAuthAuthorization renderer
func NewOAuthAuthorization(factory Factory, request model.OAuthAuthorizationRequest) (OAuthAuthorization, error) {

	const location = "render.NewOAuthAuthorization"

	// Create the result object
	result := OAuthAuthorization{
		_service: factory.OAuthApplication(),
		_app:     model.NewOAuthApplication(),
		_request: request,
	}

	// Try to load the OAuthApplication object
	if err := result._service.LoadByClientID(request.ClientID, &result._app); err != nil {
		return OAuthAuthorization{}, derp.Wrap(err, location, "Error loading OAuth Application")
	}

	// Validate the transaction
	if err := result._request.Validate(result._app); err != nil {
		return OAuthAuthorization{}, derp.Wrap(err, location, "Invalid authorization request")
	}

	// Return success.
	return result, nil
}

func (r OAuthAuthorization) ClientID() string {
	return r._app.OAuthApplicationID.Hex()
}

func (r OAuthAuthorization) Name() string {
	return r._app.Name
}

func (r OAuthAuthorization) Website() string {
	return r._app.Website
}

func (r OAuthAuthorization) RedirectURI() string {
	spew.Dump(r._request)
	return r._request.RedirectURI
}

func (r OAuthAuthorization) ResponseType() string {
	return r._request.ResponseType
}

func (r OAuthAuthorization) Scope() string {
	return r._request.Scope
}

func (r OAuthAuthorization) Scopes() []string {

	if r._request.Scope == "" {
		return []string{"read"}
	}

	return strings.Split(r._request.Scope, " ")
}
