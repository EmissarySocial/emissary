package mastodon

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/toot"
)

// mastodon_Authorizer generates a toot.Authorizer for this serverFactory.  This
// function validates the "Authorization" header, parses its JWT token, and returns a
// model.Authorization object when successful.  This function also verifies that the
// JWT token was created for a particular OAuth client and is not a regular User token
func Authorizer(serverFactory *server.Factory) toot.Authorizer[model.Authorization] {

	const location = "handler.mastodon.Authorizater"

	return func(request *http.Request) (model.Authorization, error) {

		// Get the factory for this domain
		hostname := dt.TrueHostname(request)
		factory, err := serverFactory.ByHostname(hostname)

		if err != nil {
			return model.Authorization{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Parse the JWT token from the request
		jwtService := factory.JWT()
		token, err := jwtService.Parse(request)

		if err != nil {
			return model.Authorization{}, derp.Wrap(err, location, "Invalid JWT token")
		}

		// Validate the token
		if !token.Valid {
			return model.Authorization{}, derp.ForbiddenError(location, "Invalid token: Invalid JWT")
		}

		authorization, ok := token.Claims.(*model.Authorization)

		if !ok {
			return model.Authorization{}, derp.ForbiddenError(location, "Invalid token: Invalid Claims", token)
		}

		// Return the token to the caller.
		return *authorization, nil
	}
}
