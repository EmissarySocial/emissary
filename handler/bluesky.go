package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/labstack/echo/v4"
)

func GetBlueskyDID(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetBlueskyDID"

	return func(ctx echo.Context) error {

		hostname := ctx.Request().Host

		// For development, mock a user
		if domain.IsLocalhost(hostname) {
			hostname = "example.localhost"
		}

		// Isolate the username and hostname
		username, hostname, exists := strings.Cut(hostname, ".")

		if !exists {
			return derp.NotFound(location, "Username/hostname not found")
		}

		// Get the factory for the chosen hostname
		factory, err := serverFactory.ByHostname(hostname)
		if err != nil {
			return derp.Wrap(err, location, "Invalid hostname")
		}

		// Get a new database session
		session, cancelFunc, err := factory.Session(time.Second * 5)
		if err != nil {
			return derp.Wrap(err, location, "Failed to create session")
		}
		defer cancelFunc()

		// Load the Bluesky connector configuration
		connectionService := factory.Connection()
		connection := model.NewConnection()

		if err := connectionService.LoadByProvider(session, model.ConnectionProviderBluesky, &connection); err != nil {
			return derp.Wrap(err, location, "Unable to load Bluesky configuration")
		}

		// Try to find the requested user
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByUsername(session, username, &user); err != nil {
			return derp.Wrap(err, location, "Unable to load User", username)
		}

		// RULE: Requre that the user has opted in to Bluesky bridging
		if user.IsBridgeBluesky.IsFalse() {
			return derp.Wrap(err, location, "User has not opted in to Bluesky bridging", username)
		}

		// Generate the correct Bridgy URL for this user, and forward the request there
		forwardTo := connection.Data.GetString("serverUrl") + "/.well-known/atproto-did?protocol=ap&id=" + user.ProfileURL
		return ctx.Redirect(http.StatusSeeOther, forwardTo)
	}
}
