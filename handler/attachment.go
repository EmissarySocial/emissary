package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetDomainAttachment(factoryManager *server.Factory) echo.HandlerFunc {

	const location = "handler.GetDomainAttachment"

	return func(ctx echo.Context) error {

		// Check ETags to see if the browser already has a copy of this
		if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == "1" {
			return ctx.NoContent(http.StatusNotModified)
		}

		// Get factory from the request
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Cannot load Domain")
		}

		domain := factory.Domain().Get()

		// Load the attachment in order to verify that it is valid for this stream
		// TODO: LOW: This might be more efficient as a single query...
		attachmentService := factory.Attachment()
		attachmentIDString := list.Dot(ctx.Param("attachmentId")).First()
		attachmentID, err := primitive.ObjectIDFromHex(attachmentIDString)

		if err != nil {
			return derp.Wrap(err, location, "Invalid attachmentID", attachmentIDString)
		}

		attachment := model.NewAttachment(model.AttachmentObjectTypeDomain, domain.DomainID)
		if err := attachmentService.LoadByID(model.AttachmentObjectTypeDomain, domain.DomainID, attachmentID, &attachment); err != nil {
			return derp.Wrap(err, location, "Error loading attachment")
		}

		// Retrieve the file from the mediaserver
		ms := factory.MediaServer()
		filespec := attachment.FileSpec(ctx.Request().URL)

		header := ctx.Response().Header()
		header.Set("Content-Type", filespec.MimeType())
		header.Set("ETag", "1")
		header.Set("Cache-Control", "public, max-age=86400") // Store in public caches for 1 day

		if err := ms.Serve(ctx.Response().Writer, ctx.Request(), filespec); err != nil {
			return derp.Wrap(err, location, "Error accessing attachment file", derp.WithCode(http.StatusInternalServerError))
		}

		return nil
	}
}

func GetSearchTagAttachment(ctx *steranko.Context, factory *domain.Factory) error {

	const location = "handler.GetSearchTagAttachment"

	// Check ETags to see if the browser already has a copy of this
	if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == "1" {
		return ctx.NoContent(http.StatusNotModified)
	}

	attachmentService := factory.Attachment()

	// Locate the SearchTagID
	searchTagID, err := primitive.ObjectIDFromHex(ctx.Param("searchTagId"))

	if err != nil {
		return derp.Wrap(err, location, "Invalid SearchTagID")
	}

	// Locate the AttachmentID
	attachmentID, err := primitive.ObjectIDFromHex(ctx.Param("attachmentId"))

	if err != nil {
		return derp.Wrap(err, location, "Invalid AttachmentID")
	}

	// Load the Attachment record from the database
	attachment := model.NewAttachment(model.AttachmentObjectTypeSearchTag, searchTagID)

	if err := attachmentService.LoadByID(model.AttachmentObjectTypeSearchTag, searchTagID, attachmentID, &attachment); err != nil {
		return derp.Wrap(err, location, "Error loading attachment")
	}

	// Retrieve the file from the mediaserver
	ms := factory.MediaServer()
	filespec := attachment.FileSpec(ctx.Request().URL)

	if err := ms.Serve(ctx.Response().Writer, ctx.Request(), filespec); err != nil {
		return derp.Wrap(err, location, "Error accessing attachment file", derp.WithCode(http.StatusInternalServerError))
	}

	return nil
}

func GetStreamAttachment(factoryManager *server.Factory) echo.HandlerFunc {

	const location = "handler.GetAttachment"

	return func(ctx echo.Context) error {

		// Check ETags to see if the browser already has a copy of this
		if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == "1" {
			return ctx.NoContent(http.StatusNotModified)
		}

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
		// TODO: LOW: This might be more efficient as a single query...
		attachmentService := factory.Attachment()
		attachmentIDString := list.Dot(ctx.Param("attachmentId")).First()
		attachmentID, err := primitive.ObjectIDFromHex(attachmentIDString)

		if err != nil {
			return derp.Wrap(err, location, "Invalid attachmentID", attachmentIDString)
		}

		attachment := model.NewAttachment(model.AttachmentObjectTypeStream, streamID)
		if err := attachmentService.LoadByID(model.AttachmentObjectTypeStream, streamID, attachmentID, &attachment); err != nil {
			return derp.Wrap(err, location, "Error loading attachment")
		}

		// Load Stream (to verify permissions?)
		var stream model.Stream
		streamService := factory.Stream()

		if err := streamService.LoadByID(streamID, &stream); err != nil {
			return derp.Wrap(err, location, "Error loading Stream", streamID)
		}

		// Try to find the action requested by the user.  This also enforces user permissions...
		if _, err := build.NewStreamWithoutTemplate(factory, ctx.Request(), ctx.Response(), &stream, "view"); err != nil {
			return derp.Wrap(err, location, "Cannot create builder", &stream, &attachment)
		}

		// Retrieve the file from the mediaserver
		ms := factory.MediaServer()
		filespec := attachment.FileSpec(ctx.Request().URL)

		if !stream.DefaultAllowAnonymous() {
			header := ctx.Response().Header()
			header.Set("Cache-Control", "private") // Store only in private caches for 1 day
		}

		if err := ms.Serve(ctx.Response().Writer, ctx.Request(), filespec); err != nil {
			return derp.Wrap(err, location, "Error accessing attachment file")
		}

		return nil
	}
}

func GetUserAttachment(factoryManager *server.Factory) echo.HandlerFunc {

	const location = "handler.GetUserAttachment"

	return func(ctx echo.Context) error {

		// Check ETags to see if the browser already has a copy of this
		if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == "1" {
			return ctx.NoContent(http.StatusNotModified)
		}

		// Get factory from the request
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Cannot load Domain")
		}

		// Get StreamID from the request
		userID, err := primitive.ObjectIDFromHex(ctx.Param("userId"))

		if err != nil {
			return derp.Wrap(err, location, "Invalid userID", ctx.Param("userId"))
		}

		// Load the attachment in order to verify that it is valid for this stream
		// TODO: LOW: This might be more efficient as a single query...
		attachmentService := factory.Attachment()
		attachmentIDString := list.Dot(ctx.Param("attachmentId")).First()
		attachmentID, err := primitive.ObjectIDFromHex(attachmentIDString)

		if err != nil {
			return derp.Wrap(err, location, "Invalid attachmentID", attachmentIDString)
		}

		attachment := model.NewAttachment(model.AttachmentObjectTypeUser, userID)
		if err := attachmentService.LoadByID(model.AttachmentObjectTypeUser, userID, attachmentID, &attachment); err != nil {
			return derp.Wrap(err, location, "Error loading attachment")
		}

		// Retrieve the file from the mediaserver
		ms := factory.MediaServer()
		filespec := attachment.FileSpec(ctx.Request().URL)

		if err := ms.Serve(ctx.Response().Writer, ctx.Request(), filespec); err != nil {
			return derp.Wrap(err, location, "Error accessing attachment file")
		}

		return nil
	}
}
