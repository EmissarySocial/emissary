package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetDomainAttachment(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetDomainAttachment"

	// Check ETags to see if the browser already has a copy of this
	if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == "IMMUTABLE" {
		return ctx.NoContent(http.StatusNotModified)
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
	if err := attachmentService.LoadByID(session, model.AttachmentObjectTypeDomain, domain.DomainID, attachmentID, &attachment); err != nil {
		return derp.Wrap(err, location, "Unable to load attachment")
	}

	// Retrieve the file from the mediaserver
	ms := factory.MediaServer()
	filespec := attachment.FileSpec(ctx.Request().URL)

	header := ctx.Response().Header()
	header.Set("Content-Type", filespec.MimeType())
	header.Set("ETag", "IMMUTABLE")
	header.Set("Cache-Control", "public, max-age=86400") // Store in public caches for 1 day

	if err := ms.Serve(ctx.Response().Writer, ctx.Request(), filespec); err != nil {
		return derp.Wrap(err, location, "Error accessing attachment file", derp.WithCode(http.StatusInternalServerError))
	}

	return nil
}

func GetSearchTagAttachment(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetSearchTagAttachment"

	// Check ETags to see if the browser already has a copy of this
	if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == "IMMUTABLE" {
		return ctx.NoContent(http.StatusNotModified)
	}

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
	attachmentService := factory.Attachment()
	attachment := model.NewAttachment(model.AttachmentObjectTypeSearchTag, searchTagID)

	if err := attachmentService.LoadByID(session, model.AttachmentObjectTypeSearchTag, searchTagID, attachmentID, &attachment); err != nil {
		return derp.Wrap(err, location, "Unable to load attachment")
	}

	// Retrieve the file from the mediaserver
	ms := factory.MediaServer()
	filespec := attachment.FileSpec(ctx.Request().URL)

	if err := ms.Serve(ctx.Response().Writer, ctx.Request(), filespec); err != nil {
		return derp.Wrap(err, location, "Unable to access attachment file", derp.WithCode(http.StatusInternalServerError))
	}

	return nil
}

func GetStreamAttachment(ctx *steranko.Context, factory *service.Factory, session data.Session, stream *model.Stream) error {

	const location = "handler.GetAttachment"

	// Check ETags to see if the browser already has a copy of this
	if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == "IMMUTABLE" {
		return ctx.NoContent(http.StatusNotModified)
	}

	// Try to find the action requested by the user.  This also enforces user permissions...
	if _, err := build.NewStreamWithoutTemplate(factory, session, ctx.Request(), ctx.Response(), stream, "view"); err != nil {
		return derp.Wrap(err, location, "Cannot create builder", stream)
	}

	// Load the attachment in order to verify that it is valid for this stream
	// TODO: LOW: This might be more efficient as a single query...
	attachmentService := factory.Attachment()
	attachmentToken := list.Dot(ctx.Param("attachmentId")).First()
	attachment := model.NewEmptyAttachment()
	if err := attachmentService.LoadByToken(session, model.AttachmentObjectTypeStream, stream.StreamID, attachmentToken, &attachment); err != nil {
		return derp.Wrap(err, location, "Unable to load attachment")
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

func GetUserAttachment(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.GetUserAttachment"

	// Check ETags to see if the browser already has a copy of this
	if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == "IMMUTABLE" {
		return ctx.NoContent(http.StatusNotModified)
	}

	// IF the website visitor cannot see this User, then they cannot see this User's attachments.
	if !isUserVisible(ctx, user) {
		return derp.Forbidden(location, "Cannot view attachments for an unpublished user")
	}

	// Load the attachment from the database
	attachmentService := factory.Attachment()
	token := list.Dot(ctx.Param("attachmentId")).First()
	attachment := model.NewEmptyAttachment()

	if err := attachmentService.LoadByToken(session, model.AttachmentObjectTypeUser, user.UserID, token, &attachment); err != nil {
		return derp.Wrap(err, location, "Unable to load attachment")
	}

	// Retrieve the file from the mediaserver
	ms := factory.MediaServer()
	filespec := attachment.FileSpec(ctx.Request().URL)

	if err := ms.Serve(ctx.Response().Writer, ctx.Request(), filespec); err != nil {
		return derp.Wrap(err, location, "Error accessing attachment file")
	}

	// Successfully delivered the Attachments
	return nil
}
