package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ForwardMeURLs redirects the user to their own profile page
func ForwardMeURLs(ctx *steranko.Context, factory *domain.Factory, session data.Session, user *model.User) error {
	return ctx.Redirect(http.StatusSeeOther, "/@"+user.Username)
}

// GetOutbox handles GET requests
func GetOutbox(ctx *steranko.Context, factory *domain.Factory, session data.Session, user *model.User) error {
	return buildOutbox(ctx, factory, session, user, build.ActionMethodGet)
}

// PostOutbox handles POST/DELETE requests
func PostOutbox(ctx *steranko.Context, factory *domain.Factory, session data.Session, user *model.User) error {
	return buildOutbox(ctx, factory, session, user, build.ActionMethodPost)
}

func GetProfileIcon(ctx *steranko.Context, factory *domain.Factory, session data.Session, user *model.User) error {

	filespec := mediaserver.FileSpec{
		Extension: ".webp",
		Height:    300,
		Width:     300,
	}

	return getUserAttachment(ctx, factory, session, user, "iconId", filespec)
}

func GetProfileImage(ctx *steranko.Context, factory *domain.Factory, session data.Session, user *model.User) error {

	filespec := mediaserver.FileSpec{
		Extension: ".webp",
		Width:     2400,
	}

	return getUserAttachment(ctx, factory, session, user, "imageId", filespec)
}

func PostProfileDelete(ctx *steranko.Context, factory *domain.Factory, session data.Session, user *model.User) error {

	const location = "handler.PostProfileDelete"

	// Get the request data
	values, err := formdata.Parse(ctx.Request())

	if err != nil {
		return derp.Wrap(err, location, "Error parsing form values")
	}

	if values.Get("confirm") != user.Username {
		return inlineError(ctx, `Incorrect Username. Try Again.`)
	}

	userService := factory.User()

	if err := userService.Delete(session, user, "Deleted by User"); err != nil {
		return derp.Wrap(err, "handler.PostProfileDelete", "Error deleting user")
	}

	return ctx.Redirect(http.StatusTemporaryRedirect, "/signout")
}

// buildOutbox is the common Outbox handler for both GET and POST requests
func buildOutbox(ctx *steranko.Context, factory *domain.Factory, session data.Session, user *model.User, actionMethod build.ActionMethod) error {

	const location = "handler.buildOutbox"

	// Get the UserID from the URL (could be "me")
	username, err := profileUsername(ctx)

	if err != nil {
		return derp.Wrap(err, location, "Error loading user ID")
	}

	if !isUserVisible(ctx, user) {
		return derp.NotFoundError("handler.buildOutbox", "User not found")
	}

	// Try to load the User's Outbox
	actionID := first.String(ctx.Param("action"), "view")

	// If we've directly loaded the User's profile page using a
	// hex userID then replace the URL to use their username
	// instead of their userID
	if actionID == "view" {
		if hxRequest := ctx.Request().Header.Get("Hx-Request"); hxRequest == "true" {
			if userIDHex := user.UserID.Hex(); userIDHex == username {
				if userIDHex != user.Username {
					ctx.Response().Header().Set("Hx-Replace-Url", "/@"+user.Username)
				}
			}
		}
	}

	builder, err := build.NewOutbox(factory, session, ctx.Request(), ctx.Response(), user, actionID)

	if err != nil {
		return derp.Wrap(err, location, "Error creating builder")
	}

	// Forward to the standard page builder to complete the job
	return build.AsHTML(ctx, factory, builder, actionMethod)
}

func getUserAttachment(ctx *steranko.Context, factory *domain.Factory, session data.Session, user *model.User, field string, filespec mediaserver.FileSpec) error {

	const location = "handler.outbox.getUserAttachment"

	if !isUserVisible(ctx, user) {
		return derp.NotFoundError(location, "User not found")
	}

	// Get the icon/image value from the User
	fieldValue, ok := user.GetStringOK(field)

	if !ok {
		return derp.InternalError(location, "Invalid attachment field.  This should never happen", field)
	}

	filespec.Filename = fieldValue

	// Check ETags for the User's avatar
	if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == fieldValue {
		return ctx.NoContent(http.StatusNotModified)
	}

	// Retrieve the file from the mediaserver
	ms := factory.MediaServer()
	if err := ms.Serve(ctx.Response().Writer, ctx.Request(), filespec); err != nil {
		return derp.Wrap(err, location, "Error accessing profile attachment file")
	}

	return nil
}

// profileUsername returns a string version of the UserID.
// if the username is "me" then this function returns the currently authenticated user's ID.
func profileUsername(context echo.Context) (string, error) {

	const location = "handler.profileUserID"

	userIDString := context.Param("userId")

	switch userIDString {

	case "":
		return "", derp.BadRequestError(location, "Missing UserID")

	case "me":
		userID, err := authenticatedID(context)

		if err != nil {
			return "", derp.Wrap(err, location, "Cannot use 'me' when not authenticated", derp.WithCode(http.StatusUnauthorized))
		}

		return userID.Hex(), nil

	default:
		return userIDString, nil
	}

}

// AuthenticatedID returns the UserID of the currently authenticated user.
// If the user is not signed in, then this function returns an error.
func authenticatedID(ctx echo.Context) (primitive.ObjectID, error) {

	authorization := getAuthorization(ctx)

	if authorization.IsAuthenticated() {
		return authorization.UserID, nil
	}

	return primitive.NilObjectID, derp.UnauthorizedError("handler.profileUserID", "User is not authenticated")
}
