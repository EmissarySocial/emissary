package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PostFolderReadDate(serverFactory *server.Factory) echo.HandlerFunc {

	return func(context echo.Context) error {

		const location = "handler.PostFolderReadDate"

		// User must be authenticated to use this function
		userID, err := authenticatedID(context)

		if err != nil {
			return derp.Wrap(err, location, "Error getting authenticated user ID")
		}

		// Get the folder ID from the URL
		folderID, err := primitive.ObjectIDFromHex(context.QueryParam("folderId"))

		if err != nil {
			return derp.Wrap(err, location, "Error parsing folder ID", context.Param("folderId"))
		}

		// Get the readDate from the query string
		rank := convert.Int64(context.QueryParam("rank"))

		if rank == 0 {
			return derp.Wrap(err, location, "Invalid rank", context.QueryParam("rank"))
		}

		// Get the factory for this domain
		factory, err := serverFactory.ByContext(context)

		if err != nil {
			return derp.Wrap(err, location, "Error getting server factory")
		}

		// Update the Folder with the new calculations
		folderService := factory.Folder()
		if err := folderService.CalculateUnreadCount(userID, folderID, rank); err != nil {
			return derp.Wrap(err, location, "Error setting unread count")
		}

		// No Content Necessary. But reload the folder list...
		context.Response().Header().Set("HX-Trigger", "refreshSidebar")
		return context.NoContent(http.StatusNoContent)
	}
}
