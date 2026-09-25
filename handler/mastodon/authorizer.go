package mastodon

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
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

		// Parse the JWT token from the request
		jwtService := factory.JWT()
		token, err := jwtService.Parse(request)

		if err != nil {
			return model.Authorization{}, derp.Wrap(err, location, "Invalid JWT token")
		}

		// Validate the token
		if !token.Valid {
			return model.Authorization{}, derp.Forbidden(location, "Invalid token: Invalid JWT")
		}

		authorization, ok := token.Claims.(*model.Authorization)

		if !ok {
			return model.Authorization{}, derp.Forbidden(location, "Invalid token: Invalid Claims")
		}

		// Return the token to the caller.
		return *authorization, nil
	}
}
