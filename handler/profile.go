package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/render"
	"github.com/whisperverse/whisperverse/server"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetUserList(factoryManager *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		return ctx.HTML(http.StatusOK, "<h1>User List</h1>")
	}
}

// GetProfile handles GET requests
func GetProfile(factoryManager *server.Factory) echo.HandlerFunc {
	return renderProfile(factoryManager, "social-outbox", render.ActionMethodGet)
}

// PostProfile handles POST/DELETE requests
func PostProfile(factoryManager *server.Factory) echo.HandlerFunc {
	return renderProfile(factoryManager, "social-outbox", render.ActionMethodPost)
}

// GetInbox handles GET requests
func GetInbox(factoryManager *server.Factory) echo.HandlerFunc {
	return renderProfile(factoryManager, "social-inbox", render.ActionMethodGet)
}

// PostInbox handles POST/DELETE requests
func PostInbox(factoryManager *server.Factory) echo.HandlerFunc {
	return renderProfile(factoryManager, "social-inbox", render.ActionMethodPost)
}

// renderProfile is the common Profile handler for both GET and POST requests
func renderProfile(fm *server.Factory, templateID string, actionMethod render.ActionMethod) echo.HandlerFunc {

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

		// Retrieve the StreamID to display (inbox / outbox / ???)
		var streamID primitive.ObjectID

		if templateID == "social-inbox" {
			streamID = user.InboxID
		} else {
			streamID = user.OutboxID
		}

		// Try to load the inbox for this user
		streamService := factory.Stream()
		stream := model.NewStream()

		if err := streamService.LoadByID(streamID, &stream); err != nil {
			return derp.Wrap(err, location, "Error loading Outbox", userToken)
		}

		// Try to make a renderer for this request
		actionID := getActionID(ctx)

		renderer, err := render.NewStreamWithoutTemplate(factory, sterankoContext, &stream, actionID)

		if err != nil {
			return derp.Wrap(err, location, "Error creating renderer")
		}

		// Render the page
		return renderPage(factory, sterankoContext, renderer, actionMethod)
	}
}
