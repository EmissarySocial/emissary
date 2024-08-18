package handler

import (
	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/labstack/echo/v4"
)

// GetInbox handles GET requests
func GetInbox(serverFactory *server.Factory) echo.HandlerFunc {
	return buildInbox(serverFactory, build.ActionMethodGet)
}

// PostInbox handles POST/DELETE requests
func PostInbox(serverFactory *server.Factory) echo.HandlerFunc {
	return buildInbox(serverFactory, build.ActionMethodPost)
}

// buildInbox is the common Inbox handler for both GET and POST requests
func buildInbox(serverFactory *server.Factory, actionMethod build.ActionMethod) echo.HandlerFunc {

	const location = "handler.buildInbox"

	return func(ctx echo.Context) error {

		// Get the domain factory from the context
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error loading domain factory")
		}

		// Try to load the user from the database
		userService := factory.User()
		user := model.NewUser()

		authorization := getAuthorization(ctx)

		if !authorization.IsAuthenticated() {
			return derp.NewUnauthorizedError(location, "Not Authorized")
		}

		if err := userService.LoadByID(authorization.UserID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user", authorization.UserID)
		}

		// Try to load the User's Outbox
		actionID := first.String(ctx.Param("action"), "inbox")

		if ok, err := handleJSONLD(ctx, &user); ok {
			return derp.Wrap(err, location, "Error building JSON-LD")
		}

		builder, err := build.NewInbox(factory, ctx.Request(), ctx.Response(), &user, actionID)

		if err != nil {
			return derp.Wrap(err, location, "Error creating builder")
		}

		// Forward to the standard page builder to complete the job
		return build.AsHTML(factory, ctx, builder, actionMethod)
	}
}
