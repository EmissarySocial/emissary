package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PostMessageMarkRead(serverFactory *server.Factory) echo.HandlerFunc {

	return func(context echo.Context) error {

		const location = "handler.PostFolderReadDate"

		// User must be authenticated to use this function
		userID, err := authenticatedID(context)

		if err != nil {
			return derp.Wrap(err, location, "Error getting authenticated user ID")
		}

		// Get the folder ID from the URL
		messageID, err := primitive.ObjectIDFromHex(context.Param("message"))

		if err != nil {
			return derp.Wrap(err, location, "Error parsing folder ID", context.Param("message"))
		}

		// Get the factory for this domain
		factory, err := serverFactory.ByContext(context)

		if err != nil {
			return derp.Wrap(err, location, "Error getting server factory")
		}

		// Try to mark the message as "read"
		inboxService := factory.Inbox()
		message := model.NewMessage()

		if err := inboxService.LoadByID(userID, messageID, &message); err != nil {
			return derp.Wrap(err, location, "Error loading message")
		}

		if err := inboxService.MarkRead(&message); err != nil {
			return derp.Wrap(err, location, "Error marking message read")
		}

		// No Content Necessary. But reload the folder list...
		context.Response().Header().Set("HX-Trigger", "refreshSidebar")
		return context.NoContent(http.StatusNoContent)
	}
}
