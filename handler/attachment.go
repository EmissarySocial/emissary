package handler

import (
	"mime"
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetDomainAttachment serves a file attached to the Domain record
func GetDomainAttachment(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetDomainAttachment"

	// Check ETags to see if the browser already has a copy of this
	if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == "IMMUTABLE" {
		return ctx.NoContent(http.StatusNotModified)
	}

	domain := factory.Domain().Get()

	// Load the attachment in order to verify that it is valid for this stream
	attachmentService := factory.Attachment()
	attachmentIDString := list.Dot(ctx.Param("attachmentId")).First()
	attachmentID, err := primitive.ObjectIDFromHex(attachmentIDString)

	if err != nil {
		return derp.Wrap(err, location, "Invalid attachmentID", attachmentIDString, derp.WithNotFound())
	}

	attachment := model.NewAttachment(model.AttachmentObjectTypeDomain, domain.DomainID)
	if err := attachmentService.LoadByID(session, model.AttachmentObjectTypeDomain, domain.DomainID, attachmentID, &attachment); err != nil {
		return derp.Wrap(err, location, "Loading attachment")
	}

	// Retrieve the file from the mediaserver
	ms := factory.MediaServer()
	filespec := attachment.FileSpec(ctx.Request().URL)

	setAttachmentHeaders(ctx, attachment, filespec)

	header := ctx.Response().Header()
	header.Set("ETag", "IMMUTABLE")
	header.Set("Cache-Control", "public, max-age=86400") // Store in public caches for 1 day

	if err := ms.Serve(ctx.Response().Writer, ctx.Request(), filespec); err != nil {
		return serveAttachmentError(err, location, attachment)
	}

	return nil
}

// GetSearchTagAttachment serves a file attached to a SearchTag
func GetSearchTagAttachment(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetSearchTagAttachment"

	// Check ETags to see if the browser already has a copy of this
	if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == "IMMUTABLE" {
		return ctx.NoContent(http.StatusNotModified)
	}

	// Locate the SearchTagID
	searchTagID, err := primitive.ObjectIDFromHex(ctx.Param("searchTagId"))

	if err != nil {
		return derp.Wrap(err, location, "Invalid SearchTagID", derp.WithNotFound())
	}

	// Locate the AttachmentID
	attachmentID, err := primitive.ObjectIDFromHex(ctx.Param("attachmentId"))

	if err != nil {
		return derp.Wrap(err, location, "Invalid AttachmentID", derp.WithNotFound())
	}

	// Load the Attachment record from the database
	attachmentService := factory.Attachment()
	attachment := model.NewAttachment(model.AttachmentObjectTypeSearchTag, searchTagID)

	if err := attachmentService.LoadByID(session, model.AttachmentObjectTypeSearchTag, searchTagID, attachmentID, &attachment); err != nil {
		return derp.Wrap(err, location, "Loading attachment")
	}

	// Retrieve the file from the mediaserver
	ms := factory.MediaServer()
	filespec := attachment.FileSpec(ctx.Request().URL)

	setAttachmentHeaders(ctx, attachment, filespec)

	if err := ms.Serve(ctx.Response().Writer, ctx.Request(), filespec); err != nil {
		return serveAttachmentError(err, location, attachment)
	}

	return nil
}

// GetStreamAttachment serves a file attached to a Stream
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
	attachmentService := factory.Attachment()
	attachmentToken := list.Dot(ctx.Param("attachmentId")).First()
	attachment := model.NewEmptyAttachment()
	if err := attachmentService.LoadByToken(session, model.AttachmentObjectTypeStream, stream.StreamID, attachmentToken, &attachment); err != nil {
		return derp.Wrap(err, location, "Loading attachment")
	}

	// Retrieve the file from the mediaserver
	ms := factory.MediaServer()
	filespec := attachment.FileSpec(ctx.Request().URL)

	setAttachmentHeaders(ctx, attachment, filespec)

	if !stream.DefaultAllowAnonymous() {
		header := ctx.Response().Header()
		header.Set("Cache-Control", "private") // Store only in private caches for 1 day
	}

	if err := ms.Serve(ctx.Response().Writer, ctx.Request(), filespec); err != nil {
		return serveAttachmentError(err, location, attachment)
	}

	return nil
}

// GetUserAttachment serves a file attached to a User profile
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
		return derp.Wrap(err, location, "Loading attachment")
	}

	// Retrieve the file from the mediaserver
	ms := factory.MediaServer()
	filespec := attachment.FileSpec(ctx.Request().URL)

	setAttachmentHeaders(ctx, attachment, filespec)

	if err := ms.Serve(ctx.Response().Writer, ctx.Request(), filespec); err != nil {
		return serveAttachmentError(err, location, attachment)
	}

	// Successfully delivered the Attachments
	return nil
}

// setAttachmentHeaders writes the headers that tell a browser how to treat an Attachment.
func setAttachmentHeaders(ctx *steranko.Context, attachment model.Attachment, filespec mediaserver.FileSpec) {

	// Attachments are the one place where a visitor's browser reads bytes that another
	// User chose, so the response has to describe them narrowly enough that they cannot
	// act.  `filespec` must be the same one handed to MediaServer.Serve, or these headers
	// describe a file that is not the one being sent.

	header := ctx.Response().Header()

	// Never let a browser look past the Content-Type below and guess for itself.
	header.Set("X-Content-Type-Options", "nosniff")

	// Attachments are passive data.  They never need script, subresources, or an origin.
	header.Set("Content-Security-Policy", "default-src 'none'; sandbox")

	// MediaServer re-encodes this file, so the body is freshly generated media and
	// `filespec` names the allow-listed format it will be generated in.
	if attachment.CanServeInline() {

		if contentType := filespec.MimeType(); contentType != "" {
			header.Set("Content-Type", contentType)
			return
		}
	}

	// RULE: Every other file reaches the client byte-for-byte, so it is typed as opaque
	// data and pushed into a download.  Left alone, http.ServeContent would type it from
	// the *request* URL's extension (or by sniffing it), and an uploaded HTML file would
	// run as a document on this domain.
	header.Set("Content-Type", "application/octet-stream")
	header.Set("Content-Disposition", attachmentContentDisposition(attachment.Original))
}

// attachmentContentDisposition builds a Content-Disposition header that forces a download.
func attachmentContentDisposition(filename string) string {

	filename = strings.TrimSpace(filename)

	if filename == "" {
		return "attachment"
	}

	// The filename was chosen by whoever uploaded the file, so it goes through
	// mime.FormatMediaType, which quotes and percent-encodes it (including any CRLF
	// that would otherwise forge a header).  An unencodable name comes back empty.
	result := mime.FormatMediaType("attachment", map[string]string{"filename": filename})

	if result == "" {
		return "attachment"
	}

	return result
}

// serveAttachmentError translates a mediaserver.Serve failure into an HTTP response.
// A file that cannot be processed (for instance an attachment whose stored bytes are
// not a decodable image) would otherwise surface as an HTTP 500 on every request.
// We downgrade it to a 404 so a single broken attachment does not read as a server
// fault -- the browser simply renders its normal broken-image state instead.
func serveAttachmentError(err error, location string, attachment model.Attachment) error {
	return derp.Wrap(err, location, "Serving attachment file", attachment.AttachmentID.Hex(), derp.WithNotFound())
}
