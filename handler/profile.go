package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetProfile handles GET requests
func GetProfile(serverFactory *server.Factory) echo.HandlerFunc {
	return renderProfile(serverFactory, render.ActionMethodGet)
}

// PostProfile handles POST/DELETE requests
func PostProfile(serverFactory *server.Factory) echo.HandlerFunc {
	return renderProfile(serverFactory, render.ActionMethodPost)
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
		username := ctx.Param("userId")
		user := model.NewUser()

		if err := userService.LoadByToken(username, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user", username)
		}

		// Check ETags for the user's avatar
		if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == user.ImageID.Hex() {
			return ctx.NoContent(http.StatusNotModified)
		}

		// Retrieve the file from the mediaserver
		ms := factory.MediaServer()
		filespec := mediaserver.FileSpec{
			Filename:  user.ImageID.Hex(),
			Extension: ".webp",
			MimeType:  "image/webp",
			Height:    300,
			Width:     300,
		}

		header := ctx.Response().Header()

		header.Set("Mime-Type", "image/webp")
		header.Set("ETag", user.ImageID.Hex())
		header.Set("Cache-Control", "public, max-age=86400") // Store in public caches for 1 day

		if err := ms.Get(filespec, ctx.Response().Writer); err != nil {
			return derp.Wrap(err, location, "Error accessing attachment file")
		}

		return nil
	}
}

// renderProfile is the common Profile handler for both GET and POST requests
func renderProfile(serverFactory *server.Factory, actionMethod render.ActionMethod) echo.HandlerFunc {

	const location = "handler.renderProfile"

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

		// Try to load the User's Outbox
		actionID := first.String(context.Param("action"), "view")

		if ok, err := handleJSONLD(context, &user); ok {
			return derp.Wrap(err, location, "Error rendering JSON-LD")
		}

		renderer, err := render.NewProfile(factory, sterankoContext, &user, actionID)

		if err != nil {
			return derp.Wrap(err, location, "Error creating renderer")
		}

		// Forward to the standard page renderer to complete the job
		return renderHTML(factory, sterankoContext, renderer, actionMethod)
	}
}

func profileUsername(context echo.Context) (string, error) {

	userIDString := context.Param("userId")

	if (userIDString == "me") || (userIDString == "") {
		userID, err := authenticatedID(context)
		return userID.Hex(), err
	}

	return userIDString, nil
}

func authenticatedID(context echo.Context) (primitive.ObjectID, error) {

	sterankoContext := context.(*steranko.Context)
	authorization := getAuthorization(sterankoContext)

	if authorization.IsAuthenticated() {
		return authorization.UserID, nil
	}

	return primitive.NilObjectID, derp.NewUnauthorizedError("handler.profileUserID", "User is not authenticated")
}
