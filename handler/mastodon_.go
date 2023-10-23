package handler

import (
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
)

func Mastodon(serverFactory *server.Factory) toot.API[model.Authorization] {

	return toot.API[model.Authorization]{
		Authorize: mastodon_Authorizer(serverFactory),

		PostApplication:                  mastodon_PostApplication(serverFactory),
		GetApplication_VerifyCredentials: mastodon_GetApplication_VerifyCredentials(serverFactory),
		PostStatus:                       mastodon_PostStatus(serverFactory),
		GetStatus:                        mastodon_GetStatus(serverFactory),
		DeleteStatus:                     mastodon_DeleteStatus(serverFactory),
	}
}

// mastodon_Authorizer generates a toot.Authorizer for this serverFactory.  This
// function validates the "Autorization" header, parses its JWT token, and returns a
// model.Authorization object when successful.  This function also verifies that the
// JWT token was created for a particular OAuth client and is not a regular User token
func mastodon_Authorizer(serverFactory *server.Factory) toot.Authorizer[model.Authorization] {

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

// getStreamFromURL is a convenience function that combines the following
// steps: 1) locate the domain from the provided Stream URL, 2) load the
// requested stream from the database, and 3) return the Stream and corresponding
// StreamService to the caller.
func getStreamFromURL(serverFactory *server.Factory, streamURL string) (model.Stream, *service.Stream, error) {

	const location = "handler.getStreamFromURI"

	// Parse the URL to 1) validate it's legit, and 2) extract the domain name
	parsedURL, err := url.Parse(streamURL)

	if err != nil {
		return model.Stream{}, nil, derp.Wrap(err, location, "Invalid URI")
	}

	// Get the factory for this Domain
	factory, err := serverFactory.ByDomainName(parsedURL.Host)

	if err != nil {
		return model.Stream{}, nil, derp.Wrap(err, location, "Unrecognized Domain")
	}

	// Try to load the requested Stream using its URL
	streamService := factory.Stream()
	stream := model.NewStream()

	if err := streamService.LoadByURL(streamURL, &stream); err != nil {
		return model.Stream{}, nil, derp.Wrap(err, location, "Error loading stream")
	}

	// Return values to the caller.
	return stream, streamService, nil

}
