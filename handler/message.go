package handler

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetMessage handles GET requests
func GetMessage(serverFactory *server.Factory) echo.HandlerFunc {
	return renderMessage(serverFactory, render.ActionMethodGet)
}

// PostMessage handles POST/DELETE requests
func PostMessage(serverFactory *server.Factory) echo.HandlerFunc {
	return renderMessage(serverFactory, render.ActionMethodPost)
}

// renderMessage is the common Inbox handler for both GET and POST requests
func renderMessage(serverFactory *server.Factory, actionMethod render.ActionMethod) echo.HandlerFunc {

	const location = "handler.renderMessage"

	return func(context echo.Context) error {

		// Cast the context into a steranko context (which includes authentication data)
		sterankoContext := context.(*steranko.Context)

		// Get the domain factory from the context
		factory, err := serverFactory.ByContext(sterankoContext)

		if err != nil {
			return derp.Wrap(err, location, "Error loading domain factory")
		}

		// Get the UserID from the URL (could be "me")
		authorization := getAuthorization(sterankoContext)

		if !authorization.IsAuthenticated() {
			return derp.NewUnauthorizedError(location, "Not Authorized")
		}

		// Get the MessageID from the URL
		messageID, err := primitive.ObjectIDFromHex(context.Param("message"))

		if err != nil {
			return derp.Wrap(err, location, "Invalid message ID", context.Param("message"))
		}

		// Try to load the user from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(authorization.UserID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user", authorization.UserID)
		}

		// Try to load the Message from the database
		inboxService := factory.Inbox()
		message := model.NewMessage()

		if err := inboxService.LoadByID(user.UserID, messageID, &message); err != nil {
			return derp.Wrap(err, location, "Error loading message", context.Param("message"))
		}

		// Move to previous/next sibling if requested
		if siblingType := context.QueryParam("sibling"); siblingType != "" {
			followingID := context.QueryParam("followingId")
			if sibling, err := inboxService.LoadSibling(message.FolderID, message.Rank, followingID, siblingType); err == nil {
				message = sibling
			}
		}

		// Render in JSON-LD (if requested)
		// TODO: Templates should probably use the new "view-json" action instead.
		if ok, err := handleJSONLD(context, &user); ok {
			return derp.Wrap(err, location, "Error rendering JSON-LD")
		}

		// Try to load the User's Outbox
		actionID := first.String(context.Param("action"), "view")

		// Create the new Renderer
		renderer, err := render.NewMessage(factory, sterankoContext, inboxService, &message, actionID)

		if err != nil {
			return derp.Wrap(err, location, "Error creating renderer")
		}

		// Forward to the standard page renderer to complete the job
		return renderHTML(factory, sterankoContext, renderer, actionMethod)
	}
}
