package handler

import (
	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
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

	return func(context echo.Context) error {

		// Cast the context into a steranko context (which includes authentication data)
		sterankoContext := context.(*steranko.Context)

		// Get the domain factory from the context
		factory, err := serverFactory.ByContext(sterankoContext)

		if err != nil {
			return derp.Wrap(err, location, "Error loading domain factory")
		}

		// Try to load the user from the database
		userService := factory.User()
		user := model.NewUser()

		authorization := getAuthorization(sterankoContext)

		if !authorization.IsAuthenticated() {
			return derp.NewUnauthorizedError(location, "Not Authorized")
		}

		if err := userService.LoadByID(authorization.UserID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user", authorization.UserID)
		}

		// Try to load the User's Outbox
		actionID := first.String(context.Param("action"), "inbox")

		if ok, err := handleJSONLD(context, &user); ok {
			return derp.Wrap(err, location, "Error building JSON-LD")
		}

		builder, err := build.NewInbox(factory, context.Request(), context.Response(), &user, actionID)

		if err != nil {
			return derp.Wrap(err, location, "Error creating builder")
		}

		// Forward to the standard page builder to complete the job
		return buildHTML(factory, sterankoContext, builder, actionMethod)
	}
}
