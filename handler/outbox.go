package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/builder"
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
	return buildOutbox(serverFactory, builder.ActionMethodGet)
}

// PostOutbox handles POST/DELETE requests
func PostOutbox(serverFactory *server.Factory) echo.HandlerFunc {
	return buildOutbox(serverFactory, builder.ActionMethodPost)
}

func GetProfileAvatar(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetProfileAvatar"

	return func(ctx echo.Context) error {

		// Cast the context into a steranko context (which includes authentication data)
		sterankoContext := ctx.(*steranko.Context)

		// Get the Domain factory from the context
		factory, err := serverFactory.ByContext(sterankoContext)

		if err != nil {
			return derp.Wrap(err, location, "Error loading domain factory")
		}

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()

		// Get the UserID from the URL (could be "me")
		username, err := profileUsername(sterankoContext)

		if err != nil {
			return derp.Wrap(err, location, "Error loading user ID")
		}

		if err := userService.LoadByToken(username, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user", username)
		}

		if !isUserVisible(sterankoContext, &user) {
			return derp.NewNotFoundError("handler.GetProfileAvatar", "User not found")
		}

		// Check ETags for the user's avatar
		if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == user.IconID.Hex() {
			return ctx.NoContent(http.StatusNotModified)
		}

		// Retrieve the file from the mediaserver
		ms := factory.MediaServer()
		filespec := mediaserver.FileSpec{
			Filename:  user.IconID.Hex(),
			Extension: ".webp",
			MimeType:  "image/webp",
			Height:    300,
			Width:     300,
		}

		header := ctx.Response().Header()

		header.Set("Mime-Type", "image/webp")
		header.Set("ETag", user.IconID.Hex())
		header.Set("Cache-Control", "public, max-age=86400") // Store in public caches for 1 day

		if err := ms.Get(filespec, ctx.Response().Writer); err != nil {
			return derp.Wrap(err, location, "Error accessing attachment file")
		}

		return nil
	}
}

// buildOutbox is the common Outbox handler for both GET and POST requests
func buildOutbox(serverFactory *server.Factory, actionMethod builder.ActionMethod) echo.HandlerFunc {

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

		builder, err := builder.NewOutbox(factory, context.Request(), context.Response(), &user, actionID)

		if err != nil {
			return derp.Wrap(err, location, "Error creating builder")
		}

		// Forward to the standard page builder to complete the job
		return buildHTML(factory, sterankoContext, builder, actionMethod)
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
func authenticatedID(context echo.Context) (primitive.ObjectID, error) {

	sterankoContext := context.(*steranko.Context)
	authorization := getAuthorization(sterankoContext)

	if authorization.IsAuthenticated() {
		return authorization.UserID, nil
	}

	return primitive.NilObjectID, derp.NewUnauthorizedError("handler.profileUserID", "User is not authenticated")
}
