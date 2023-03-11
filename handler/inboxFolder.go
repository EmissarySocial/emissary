package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PostInboxFolderReadDate(serverFactory *server.Factory) echo.HandlerFunc {

	return func(context echo.Context) error {

		// User must be authenticated to use this function
		authenticatedID, err := authenticatedID(context)

		if err != nil {
			return derp.Wrap(err, "handler.PostFolderReadDate", "Error getting authenticated user ID")
		}

		// Get the folder ID from the URL
		folderID, err := primitive.ObjectIDFromHex(context.QueryParam("folderId"))

		if err != nil {
			return derp.Wrap(err, "handler.PostFolderReadDate", "Error parsing folder ID", context.Param("folderId"))
		}

		// Get the readDate from the query string
		readDate := convert.Int64(context.QueryParam("readDate"))

		// Get the factory for this domain
		factory, err := serverFactory.ByContext(context)

		if err != nil {
			return derp.Wrap(err, "handler.PostFolderReadDate", "Error getting server factory")
		}

		// Update the read date for this folder.
		folderService := factory.Folder()

		if err := folderService.UpdateReadDate(authenticatedID, folderID, readDate); err != nil {
			return derp.Wrap(err, "handler.PostFolderReadDate", "Error updating read date", folderID, authenticatedID, readDate)
		}

		// No Content Necessary.
		return context.NoContent(http.StatusNoContent)
	}
}
