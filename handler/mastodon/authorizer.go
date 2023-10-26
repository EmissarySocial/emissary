package mastodon

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
)

// mastodon_Authorizer generates a toot.Authorizer for this serverFactory.  This
// function validates the "Autorization" header, parses its JWT token, and returns a
// model.Authorization object when successful.  This function also verifies that the
// JWT token was created for a particular OAuth client and is not a regular User token
func Authorizer(serverFactory *server.Factory) toot.Authorizer[model.Authorization] {

	const location = "handler.mastodon_Authorization"

	return func(request *http.Request) (model.Authorization, error) {

		// Get the factory for this domain
		factory, err := serverFactory.ByDomainName(request.Host)

		if err != nil {
			return model.Authorization{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Parse the JWT token from the request
		jwtService := factory.JWT()
		token, err := jwtService.Parse(request)

		if err != nil {
			return model.Authorization{}, derp.Wrap(err, location, "Invalid JWT token")
		}

		// Extract the Authorization from the JWT Token
		result := token.Claims.(model.Authorization)

		// Validate the token
		if !token.Valid {
			return result, derp.NewForbiddenError(location, "Invalid token: Invalid JWT")
		}

		// Confirm that the UserID is present
		if result.UserID.IsZero() {
			return model.Authorization{}, derp.NewForbiddenError(location, "Invalid token: missing UserID")
		}

		// Confirm that the ClientID is not empty.  This confirms
		// we have an OAuth token, not a user token.
		if result.ClientID.IsZero() {
			return result, derp.NewForbiddenError(location, "Token must be an OAuth token, not a user token")
		}

		// Return the token to the caller.
		return result, nil
	}
}
