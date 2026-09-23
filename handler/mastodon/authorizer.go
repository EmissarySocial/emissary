package mastodon

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
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
		hostname := serverFactory.Hostname(request)
		factory, err := serverFactory.ByHostname(hostname)

		if err != nil {
			return model.Authorization{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Parse the JWT token from the request. A parse failure here is an expired,
		// malformed, or badly-signed bearer token -- that is a 401, not a 500. The
		// Mastodon client only refreshes its access token when it sees a 401; any
		// other status leaves it stuck on the dead token until the user re-signs in.
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

		// Return the token to the caller.
		return *authorization, nil
	}
}
