package mastodon

import (
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Authorizer returns a toot.Authorizer that parses the JWT in a request's "Authorization"
// header, and returns the model.Authorization it carries
func Authorizer(serverFactory *server.Factory) toot.Authorizer[model.Authorization] {

	const location = "handler.mastodon.Authorizater"

	return func(request *http.Request) (model.Authorization, error) {

		// Get the factory for this domain
		hostname := serverFactory.Hostname(request)
		factory, err := serverFactory.ByHostname(hostname)

		if err != nil {
			return model.Authorization{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Parse the JWT from the request. An expired, malformed or badly signed token is a 401, not a
		// 500: Mastodon apps only react to a 401, and any other status leaves them stuck on a dead token.
		jwtService := factory.JWT()
		token, err := jwtService.Parse(request)

		if err != nil {
			return model.Authorization{}, derp.Unauthorized(location, "Invalid bearer token", err)
		}

		// Validate the token
		if !token.Valid {
			return model.Authorization{}, derp.Unauthorized(location, "Invalid bearer token: failed validation")
		}

		authorization, ok := token.Claims.(*model.Authorization)

		if !ok {
			return model.Authorization{}, derp.Unauthorized(location, "Invalid bearer token: unrecognized claims")
		}

		// RULE: The grant behind the token must still exist, so disconnecting an app takes effect at once
		if err := requireGrant(*authorization, grantLoader(factory)); err != nil {
			return model.Authorization{}, derp.Wrap(err, location, "Checking OAuth grant")
		}

		// Return the token to the caller.
		return *authorization, nil
	}
}

// requireGrant returns a 401 when the token carries no grant or its grant is gone (disconnected
// or revoked), and passes any other loading error through
func requireGrant(authorization model.Authorization, loadGrant func(userID primitive.ObjectID, grantID primitive.ObjectID) error) error {

	const location = "handler.mastodon.requireGrant"

	// RULE: Mastodon API tokens are always issued from an OAuth grant
	if authorization.OAuthUserTokenID.IsZero() {
		return derp.Unauthorized(location, "Bearer token is not an OAuth grant")
	}

	// RULE: Only a missing grant signs the app out; a database error must not look like a revoke
	if err := loadGrant(authorization.UserID, authorization.OAuthUserTokenID); err != nil {

		if derp.IsNotFound(err) {
			return derp.Unauthorized(location, "OAuth grant has been revoked", authorization.OAuthUserTokenID.Hex())
		}

		return derp.Wrap(err, location, "Loading OAuth grant", authorization.OAuthUserTokenID.Hex())
	}

	// Still connected. Carry on.
	return nil
}

// grantLoader returns a function that loads one User's OAuth grant from this domain's database
func grantLoader(factory *service.Factory) func(userID primitive.ObjectID, grantID primitive.ObjectID) error {

	const location = "handler.mastodon.grantLoader"

	return func(userID primitive.ObjectID, grantID primitive.ObjectID) error {

		session, cancel, err := factory.Session(10 * time.Second)

		if err != nil {
			return derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		grant := model.NewOAuthUserToken()
		return factory.OAuthUserToken().LoadByID(session, userID, grantID, &grant)
	}
}
