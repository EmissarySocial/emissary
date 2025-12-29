package handler

import (
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PostExportStart is a part of the Data Portability process.  It is called by the "target" server when it
// begins a migration, in order to tell the "source" server where the exported data is going -- passing the
// `actor` and `oracle` values to the "source" server for use at the end of the process.
func PostUserExportStart(ctx *steranko.Context, factory *service.Factory, session data.Session, oauthUserToken *model.OAuthUserToken, user *model.User) error {

	const location = "handler.PostUserExportStart"

	// Define the parameters we're expecting to receive from the client
	txn := struct {
		Actor  string `form:"actor"`  // The TARGET actor that is receiving this export
		Oracle string `form:"oracle"` // The oracle where we can look up object URLs after they've been exported
	}{}

	// Collect parameters from form post
	if err := ctx.Bind(&txn); err != nil {
		return derp.Wrap(err, location, "Unable to parse request")
	}

	// Populate the OAuthUserToken Data with values from the client
	oauthUserToken.Data.SetString("actor", txn.Actor)
	oauthUserToken.Data.SetString("oracle", txn.Oracle)
	oauthUserToken.Data.SetInt64("startDate", time.Now().Unix())

	// Save the updated OAuthUserToken
	oauthUserTokenService := factory.OAuthUserToken()
	if err := oauthUserTokenService.Save(session, oauthUserToken, "Starting Export via OAuth"); err != nil {
		return derp.Wrap(err, location, "Unable to save OAuthUserToken", "oauthUserTokenID", oauthUserToken.OAuthUserTokenID)
	}

	// Return an empty 200 OK response
	return ctx.NoContent(http.StatusOK)
}

// GetUserExportCollection is a part of the Data Portability process.  It retrieves a single collection
func GetUserExportCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, oauthUserToken *model.OAuthUserToken, user *model.User) error {

	const location = "handler.GetUserExportCollection"

	requestURL := fullURL(factory, ctx)

	// Locate the service to use
	exportService := factory.Export()
	collection := ctx.Param("collection")
	service, err := exportService.FindService(collection)

	// If we can't locate the service, then just return a 404 with an empty collection
	if err != nil {
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		return ctx.JSON(http.StatusNotFound, streams.NewOrderedCollection(requestURL))
	}

	// Generate the export collection for this service
	records, err := service.ExportCollection(session, user.UserID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to retrieve exportable records", collection)
	}

	// Return the result to the caller as a JSON-LD Collection
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	result := streams.NewOrderedCollection(requestURL)
	result.TotalItems = len(records)
	result.OrderedItems = slice.Map(records, func(recordID model.IDOnly) any {
		return requestURL + "/" + recordID.ID.Hex()
	})

	return ctx.JSON(http.StatusOK, result)
}

func GetUserExportDocument(ctx *steranko.Context, factory *service.Factory, session data.Session, oauthUserToken *model.OAuthUserToken, user *model.User) error {

	const location = "handler.GetUserExportDocument"

	// Locate the service to use
	exportService := factory.Export()
	collection := ctx.Param("collection")
	service, err := exportService.FindService(collection)

	if err != nil {
		return derp.Wrap(err, location, "Unable to find export service", collection)
	}

	recordID, err := primitive.ObjectIDFromHex(ctx.Param("recordId"))

	if err != nil {
		return derp.Wrap(err, location, "Invalid RecordID", ctx.Param("recordId"))
	}

	// Generate the export collection for this service
	record, err := service.ExportDocument(session, user.UserID, recordID)

	if err != nil {
		return derp.Wrap(err, location, "Uable to retrieve exportable record", collection, recordID)
	}

	// Return the result to the caller as a JSON-LD Collection
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.String(http.StatusOK, record)

}

// GetAttachmentsExportDocument returns an OrderedCollection of all
// Attachments associated with the provided objectType and objectID.
func GetAttachmentsExportCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, oauthUserToken *model.OAuthUserToken, user *model.User, stream *model.Stream) error {

	const location = "handler.GetAttachmentsExportCollection"

	// Generate the export collection for this service
	records, err := factory.Attachment().ExportCollection(session, model.AttachmentObjectTypeStream, stream.StreamID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to retrieve exportable attachments")
	}

	// Return the result to the caller as a JSON-LD Collection
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	requestURL := fullURL(factory, ctx)
	result := streams.NewOrderedCollection(requestURL)
	result.TotalItems = len(records)
	result.OrderedItems = slice.Map(records, func(recordID model.IDOnly) any {
		return requestURL + "/" + recordID.ID.Hex()
	})

	// Return JSON
	return ctx.JSON(http.StatusOK, result)
}

// GetAttachmentsExportDocument retrieves a single Attachment as a JSON string
func GetAttachmentsExportDocument(ctx *steranko.Context, factory *service.Factory, session data.Session, oauthUserToken *model.OAuthUserToken, user *model.User, stream *model.Stream) error {

	const location = "handler.GetAttachmentsExportDocument"

	// Collect the AttachmentID from the URL
	attachmentID, err := primitive.ObjectIDFromHex(ctx.Param("attachmentId"))

	if err != nil {
		return derp.Wrap(err, location, "AttachmentID must be a valid ObjectID", ctx.Param("attachmentId"))
	}

	// Generate the export collection for this service
	attachmentService := factory.Attachment()
	attachmentJSON, err := attachmentService.ExportDocument(session, model.AttachmentObjectTypeStream, stream.StreamID, attachmentID)

	if err != nil {
		return derp.Wrap(err, location, "Uable to retrieve exportable Attachment", attachmentID)
	}

	// Return the result to the caller as a JSON-LD Collection
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.String(http.StatusOK, attachmentJSON)
}

// GetAttachmentsExportOriginal serves the original file associated with the Attachment via HTTP
func GetAttachmentsExportOriginal(ctx *steranko.Context, factory *service.Factory, session data.Session, oauthUserToken *model.OAuthUserToken, user *model.User, stream *model.Stream) error {

	const location = "handler.GetAttachmentsExportDocument"

	// Collect the AttachmentID from the URL
	attachmentID, err := primitive.ObjectIDFromHex(ctx.Param("attachmentId"))

	if err != nil {
		return derp.Wrap(err, location, "Import record ID must be a valid ObjectID", ctx.Param("attachmentId"))
	}

	// Serve the original file directly to the HTTP response writer
	attachmentService := factory.Attachment()
	if err := attachmentService.ExportOriginal(session, model.AttachmentObjectTypeStream, stream.StreamID, attachmentID, ctx.Request(), ctx.Response().Writer); err != nil {
		return derp.Wrap(err, location, "Uable to serve original file", attachmentID)
	}

	// Done.
	return nil
}

// PostUserExportFinish is a part of the Data Portability process.  It initiates
// the background process to `MOVE` a user to their new server, and sign them out of this server.
func PostUserExportFinish(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.PostUserExportFinish"

	// RULE: Validate the UserID matches the authenticated user
	if ctx.Param("userId") != user.UserID.Hex() {
		return derp.Forbidden(location, "Forbidden from finishing export for another user", "url userId: "+ctx.Param("userId"), "authenticated userId: "+user.UserID.Hex())
	}

	// Parse the OAuthUserTokenID
	oauthUserTokenID, err := primitive.ObjectIDFromHex(ctx.QueryParam("oauthUserTokenId"))

	if err != nil {
		return derp.Wrap(err, location, "OAuthUserTokenID must be a valid ObjectID", ctx.Param("oauthUserTokenId"))
	}

	// Load the OAuthUserToken
	oauthUserTokenService := factory.OAuthUserToken()
	oauthUserToken := model.NewOAuthUserToken()

	if err := oauthUserTokenService.LoadByID(session, user.UserID, oauthUserTokenID, &oauthUserToken); err != nil {
		return derp.Wrap(err, location, "Unable to load OAuthUserToken", oauthUserTokenID)
	}

	actor := oauthUserToken.Data.GetString("actor")
	oracle := oauthUserToken.Data.GetString("oracle")

	// Mark the User as "Moved"
	if err := factory.User().Move(session, user, actor, oracle); err != nil {
		return derp.Wrap(err, location, "Unable to mark User as 'Moved'", user, actor)
	}

	// Sign the user out of this website.
	factory.Steranko(session).SignOut(ctx)

	// Return an empty 200 OK response that redirectst he browser to the signout page
	ctx.Response().Header().Set("HX-Redirect", "/signout")
	return ctx.NoContent(http.StatusOK)
}

// This displays a message to users that their profile has been exported.
func GetUserExportComplete(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	b := html.New()
	b.HTML()
	b.Body()
	b.H1().InnerText("Your Profile Has Been Moved, and You Have Been Signed Out.")
	b.Button().Attr("onclick", "window.close()").InnerText("Close this Window")
	b.CloseAll()

	return ctx.HTML(http.StatusOK, b.String())
}
