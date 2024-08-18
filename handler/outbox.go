package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/build"
	activitypub "github.com/EmissarySocial/emissary/handler/activitypub_user"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetOutbox handles GET requests
func GetOutbox(serverFactory *server.Factory) echo.HandlerFunc {
	return buildOutbox(serverFactory, build.ActionMethodGet)
}

// PostOutbox handles POST/DELETE requests
func PostOutbox(serverFactory *server.Factory) echo.HandlerFunc {
	return buildOutbox(serverFactory, build.ActionMethodPost)
}

func GetProfileIcon(serverFactory *server.Factory) echo.HandlerFunc {

	filespec := mediaserver.FileSpec{
		Extension: ".webp",
		MimeType:  "image/webp",
		Height:    300,
		Width:     300,
	}

	return getProfileAttachment(serverFactory, "iconId", filespec)
}

func GetProfileImage(serverFactory *server.Factory) echo.HandlerFunc {

	filespec := mediaserver.FileSpec{
		Extension: ".webp",
		MimeType:  "image/webp",
		Width:     2400,
	}

	return getProfileAttachment(serverFactory, "imageId", filespec)
}

func getProfileAttachment(serverFactory *server.Factory, field string, filespec mediaserver.FileSpec) echo.HandlerFunc {

	const location = "handler.outbox.getProfileAttachment"

	return func(ctx echo.Context) error {

		// Cast the context into a steranko context (which includes authentication data)
		sterankoContext := ctx.(*steranko.Context)

		// Get the Domain factory from the context
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error loading domain factory")
		}

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()

		// Get the UserID from the URL (could be "me")
		username, err := profileUsername(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error loading user ID")
		}

		if err := userService.LoadByToken(username, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user", username)
		}

		if !isUserVisible(sterankoContext, &user) {
			return derp.NewNotFoundError(location, "User not found")
		}

		// Get the icon/image value from the User
		fieldValue, ok := user.GetStringOK(field)

		if !ok {
			return derp.New(derp.CodeInternalError, location, "Invalid attachment field.  This should never happen", field)
		}

		filespec.Filename = fieldValue

		// Check ETags for the User's avatar
		if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == fieldValue {
			return ctx.NoContent(http.StatusNotModified)
		}

		// Retrieve the file from the mediaserver
		ms := factory.MediaServer()

		header := ctx.Response().Header()

		header.Set("Mime-Type", "image/webp")
		header.Set("ETag", fieldValue)
		header.Set("Cache-Control", "public, max-age=86400") // Store in public caches for 1 day

		if err := ms.Get(filespec, ctx.Response().Writer); err != nil {
			return derp.Wrap(err, location, "Error accessing profile attachment file")
		}

		return nil
	}
}

// buildOutbox is the common Outbox handler for both GET and POST requests
func buildOutbox(serverFactory *server.Factory, actionMethod build.ActionMethod) echo.HandlerFunc {

	const location = "handler.buildOutbox"

	return func(context echo.Context) error {

		// Cast the context into a steranko context (which includes authentication data)
		sterankoContext := context.(*steranko.Context)

		// Get the domain factory from the context
		factory, err := serverFactory.ByContext(sterankoContext)

		if err != nil {
			return derp.Wrap(err, location, "Error loading domain factory")
		}

		// Get the UserID from the URL (could be "me")
		username, err := profileUsername(sterankoContext)

		if err != nil {
			return derp.Wrap(err, location, "Error loading user ID")
		}

		// Try to load the user from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByToken(username, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user", username)
		}

		if !isUserVisible(sterankoContext, &user) {
			return derp.NewNotFoundError("handler.buildOutbox", "User not found")
		}

		if isJSONLDRequest(sterankoContext) {
			return activitypub.RenderProfileJSONLD(context, factory, &user)
		}

		// Try to load the User's Outbox
		actionID := first.String(context.Param("action"), "view")

		if ok, err := handleJSONLD(context, &user); ok {
			return derp.Wrap(err, location, "Error building JSON-LD")
		}

		builder, err := build.NewOutbox(factory, context.Request(), context.Response(), &user, actionID)

		if err != nil {
			return derp.Wrap(err, location, "Error creating builder")
		}

		// Forward to the standard page builder to complete the job
		return build.AsHTML(factory, sterankoContext, builder, actionMethod)
	}
}

// profileUsername returns a string version of the UserID.
// if the username is "me" then this function returns the currently authenticated user's ID.
func profileUsername(context echo.Context) (string, error) {

	userIDString := context.Param("userId")

	if (userIDString == "me") || (userIDString == "") {
		userID, err := authenticatedID(context)
		return userID.Hex(), err
	}

	return userIDString, nil
}

// AuthenticatedID returns the UserID of the currently authenticated user.
// If the user is not signed in, then this function returns an error.
func authenticatedID(ctx echo.Context) (primitive.ObjectID, error) {

	authorization := getAuthorization(ctx)

	if authorization.IsAuthenticated() {
		return authorization.UserID, nil
	}

	return primitive.NilObjectID, derp.NewUnauthorizedError("handler.profileUserID", "User is not authenticated")
}
