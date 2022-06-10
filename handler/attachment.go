package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/list"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/render"
	"github.com/whisperverse/whisperverse/server"
)

func GetAttachment(factoryManager *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "handler.GetAttachment", "Cannot load Domain")
		}

		var stream model.Stream
		streamService := factory.Stream()
		streamToken := getStreamToken(ctx)

		if err := streamService.LoadByToken(streamToken, &stream); err != nil {
			return derp.Wrap(err, "handler.GetAttachment", "Error loading Stream", streamToken)
		}

		// Try to find the action requested by the user.  This also enforces user permissions...
		sterankoContext := ctx.(*steranko.Context)
		if _, err := render.NewStreamWithoutTemplate(factory, sterankoContext, &stream, "view"); err != nil {
			return derp.Wrap(err, "handler.GetAttachment", "Cannot create renderer")
		}

		// Load the attachment in order to verify that it is valid for this stream
		// TODO: This might be more efficient as a single query...
		attachmentService := factory.Attachment()
		attachment, err := attachmentService.LoadByToken(stream.StreamID, list.Head(ctx.Param("attachment"), "."))

		if err != nil {
			return derp.Wrap(err, "handler.GetAttachment", "Error loading attachment")
		}

		// Check ETags to see if the browser already has a copy of this
		if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader != "" {

			if attachment.ETag() == matchHeader {
				return ctx.NoContent(http.StatusNotModified)
			}
		}

		// Retrieve the file from the mediaserver
		ms := factory.MediaServer()
		filespec := ms.FileSpec(ctx.Request().URL, attachment.DownloadExtension())

		header := ctx.Response().Header()

		header.Set("Mime-Type", attachment.DownloadMimeType())
		header.Set("ETag", attachment.ETag())

		if err := ms.Get(filespec, ctx.Response().Writer); err != nil {
			return derp.Wrap(err, "handler.GetAttachment", "Error accessing attachment file")
		}

		return nil
	}
}
