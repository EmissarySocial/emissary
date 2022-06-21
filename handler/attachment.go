package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/render"
	"github.com/whisperverse/whisperverse/server"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetAttachment(factoryManager *server.Factory) echo.HandlerFunc {

	const location = "handler.GetAttachment"

	return func(ctx echo.Context) error {

		// Get factory from the request
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Cannot load Domain")
		}

		// Get StreamID from the request
		streamID, err := primitive.ObjectIDFromHex(ctx.Param("stream"))

		if err != nil {
			return derp.Wrap(err, location, "Invalid streamID", ctx.Param("stream"))
		}

		// Load the attachment in order to verify that it is valid for this stream
		// TODO: This might be more efficient as a single query...
		attachmentService := factory.Attachment()
		attachment, err := attachmentService.LoadByToken(streamID, list.Dot(ctx.Param("attachment")).Head())

		if err != nil {
			return derp.Wrap(err, location, "Error loading attachment")
		}

		// Check ETags to see if the browser already has a copy of this
		if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader != "" {

			if attachment.ETag() == matchHeader {
				return ctx.NoContent(http.StatusNotModified)
			}
		}

		// Load Stream (to verify permissions?)
		var stream model.Stream
		streamService := factory.Stream()

		if err := streamService.LoadByID(streamID, &stream); err != nil {
			return derp.Wrap(err, location, "Error loading Stream", streamID)
		}

		// Try to find the action requested by the user.  This also enforces user permissions...
		sterankoContext := ctx.(*steranko.Context)
		if _, err := render.NewStreamWithoutTemplate(factory, sterankoContext, &stream, "view"); err != nil {
			return derp.Wrap(err, location, "Cannot create renderer")
		}

		// Retrieve the file from the mediaserver
		ms := factory.MediaServer()
		filespec := ms.FileSpec(ctx.Request().URL, attachment.DownloadExtension())

		header := ctx.Response().Header()

		header.Set("Mime-Type", attachment.DownloadMimeType())
		header.Set("ETag", attachment.ETag())

		if stream.DefaultAllowAnonymous() {
			header.Set("Cache-Control", "public, max-age=86400") // Store in public caches for 1 day
		} else {
			header.Set("Cache-Control", "private") // Store only in private caches for 1 day
		}

		if err := ms.Get(filespec, ctx.Response().Writer); err != nil {
			return derp.Wrap(err, location, "Error accessing attachment file")
		}

		return nil
	}
}
