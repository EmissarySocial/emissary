package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/render"
	"github.com/whisperverse/whisperverse/server"
)

// GetProfile handles GET requests
func GetProfile(factoryManager *server.Factory) echo.HandlerFunc {
	return renderProfile(factoryManager, "", render.ActionMethodGet)
}

// PostProfile handles POST/DELETE requests
func PostProfile(factoryManager *server.Factory) echo.HandlerFunc {
	return renderProfile(factoryManager, "", render.ActionMethodPost)
}

// GetInbox handles GET requests
func GetInbox(factoryManager *server.Factory) echo.HandlerFunc {
	return renderProfile(factoryManager, "inbox", render.ActionMethodGet)
}

// PostInbox handles POST/DELETE requests
func PostInbox(factoryManager *server.Factory) echo.HandlerFunc {
	return renderProfile(factoryManager, "inbox", render.ActionMethodPost)
}

// GetOutbox handles GET requests
func GetOutbox(factoryManager *server.Factory) echo.HandlerFunc {
	return renderProfile(factoryManager, "outbox", render.ActionMethodGet)
}

// PostOutbox handles POST/DELETE requests
func PostOutbox(factoryManager *server.Factory) echo.HandlerFunc {
	return renderProfile(factoryManager, "outbox", render.ActionMethodPost)
}

// renderProfile is the common Profile handler for both GET and POST requests
func renderProfile(fm *server.Factory, actionID string, actionMethod render.ActionMethod) echo.HandlerFunc {

	const location = "handler.renderProfile"

	return func(ctx echo.Context) error {

		// Guarantee that the user is signed in to view this page.
		sterankoContext := ctx.(*steranko.Context)
		authorization := getAuthorization(sterankoContext)

		if !authorization.IsAuthenticated() {
			return ctx.NoContent(http.StatusUnauthorized)
		}

		// Get the user token from a) the URL, or b) the authentication cookie
		userToken := ctx.Param("user")

		if userToken == "" {
			userToken = authorization.UserID.Hex()
		}

		// If actionID has not been set for us, then get it from the context
		if actionID == "" {
			actionID = getActionID(ctx)
		}

		spew.Dump("renderProfile", userToken, actionID)

		// Try to locate the domain from the Context
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Try to load the user profile by userId or username.
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByToken(userToken, &user); err != nil {
			return derp.Wrap(err, location, "Error loading User", userToken)
		}

		// Create a profile renderer
		renderer, err := render.NewProfile(factory, sterankoContext, &user, actionID)

		if err != nil {
			return derp.Wrap(err, location, "Error creating renderer", userToken, actionID)
		}

		// Render the resulting page.
		return renderPage(factory, sterankoContext, renderer, actionMethod)
	}
}
