package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/EmissarySocial/emissary/tools/headers"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ForwardMeURLs redirects the user to their own profile page
func ForwardMeURLs(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {
	return ctx.Redirect(http.StatusSeeOther, "/@"+user.Username)
}

// HeadOutbox handles HEAD requests
func HeadOutbox(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	if !isUserVisible(ctx, user) {
		return derp.NotFound("handler.HeadOutbox", "User not found")
	}

	// RULE: HEAD must report the headers the equivalent GET would send (RFC 9110 s9.3.2). Both verbs
	// derive them from `headers`, so neither can drift away from the other.
	headers.SetAll(ctx.Response().Header(), headers.VariantOf(ctx.Request()), user)

	return ctx.NoContent(http.StatusOK)
}

// GetOutbox handles GET requests
func GetOutbox(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {
	return buildOutbox(ctx, factory, session, user, build.ActionMethodGet)
}

// PostOutbox handles POST/DELETE requests
func PostOutbox(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {
	return buildOutbox(ctx, factory, session, user, build.ActionMethodPost)
}

// GetProfileIcon serves the User's avatar image, resized to a square thumbnail
func GetProfileIcon(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	filespec := mediaserver.FileSpec{
		Extension: ".webp",
		Height:    300,
		Width:     300,
	}

	return getUserAttachment(ctx, factory, user, "iconId", filespec)
}

// GetProfileImage serves the User's banner image, resized to a maximum width
func GetProfileImage(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	filespec := mediaserver.FileSpec{
		Extension: ".webp",
		Width:     2400,
	}

	return getUserAttachment(ctx, factory, user, "imageId", filespec)
}

// PostProfileDelete deletes the signed-in User's own account, and signs them out
func PostProfileDelete(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.PostProfileDelete"

	// Get the request data
	values, err := formdata.Parse(ctx.Request())

	if err != nil {
		return derp.Wrap(err, location, "Parsing form values")
	}

	// RULE: The User must retype their own username to confirm this is deliberate
	if values.Get("confirm") != user.Username {
		return inlineError(ctx, `Incorrect Username. Try Again.`)
	}

	// Delete the User record
	userService := factory.User()

	if err := userService.Delete(session, user, "Deleted by User"); err != nil {
		return derp.Wrap(err, "handler.PostProfileDelete", "Deleting user")
	}

	// Clear the (now deleted) user's authentication cookie.  GET /signout only displays
	// a confirmation page, so the sign-out must happen here.
	factory.Steranko(session).SignOut(ctx)

	return ctx.Redirect(http.StatusTemporaryRedirect, "/signout")
}

// buildOutbox is the common Outbox handler for both GET and POST requests
func buildOutbox(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User, actionMethod build.ActionMethod) error {

	const location = "handler.buildOutbox"

	// Get the UserID from the URL (could be "me")
	username, err := profileUsername(ctx)

	if err != nil {
		return derp.Wrap(err, location, "Loading user ID")
	}

	if !isUserVisible(ctx, user) {
		return derp.NotFound("handler.buildOutbox", "User not found")
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
		return derp.Wrap(err, location, "Creating builder")
	}

	// Forward to the standard page builder to complete the job
	return build.AsHTML(ctx, factory, builder, actionMethod)
}

// getUserAttachment serves the image named by a User field (icon or banner) through the mediaserver
func getUserAttachment(ctx *steranko.Context, factory *service.Factory, user *model.User, field string, filespec mediaserver.FileSpec) error {

	const location = "handler.outbox.getUserAttachment"

	// RULE: A non-public User's images are not public
	if !isUserVisible(ctx, user) {
		return derp.NotFound(location, "User not found")
	}

	// Get the icon/image value from the User
	fieldValue, ok := user.GetStringOK(field)

	if !ok {
		return derp.Internal(location, "Invalid attachment field.  This should never happen", field)
	}

	filespec.Filename = fieldValue

	// Check ETags for the User's avatar
	if matchHeader := ctx.Request().Header.Get("If-None-Match"); matchHeader == fieldValue {
		return ctx.NoContent(http.StatusNotModified)
	}

	// Retrieve the file from the mediaserver
	ms := factory.MediaServer()
	if err := ms.Serve(ctx.Response().Writer, ctx.Request(), filespec); err != nil {
		return derp.Wrap(err, location, "Accessing profile attachment file")
	}

	return nil
}

// profileUsername returns a string version of the UserID.
// if the username is "me" then this function returns the currently authenticated user's ID.
func profileUsername(context echo.Context) (string, error) {

	const location = "handler.profileUserID"

	userIDstring := context.Param("userId")

	switch userIDstring {

	// RULE: userID must not be empty
	case "":
		return "", derp.BadRequest(location, "Missing UserID")

	// If userID is "me", then return the currently authenticated user's ID
	case "me":
		userID, err := authenticatedID(context)

		if err != nil {
			return "", derp.Wrap(err, location, "Cannot use 'me' when not authenticated", derp.WithUnauthorized())
		}

		return userID.Hex(), nil
	}

	// Otherwise, usethe userID from the URL
	return userIDstring, nil
}

// AuthenticatedID returns the UserID of the currently authenticated user.
// If the user is not signed in, then this function returns an error.
func authenticatedID(ctx echo.Context) (primitive.ObjectID, error) {

	if authorization := getAuthorization(ctx); authorization.IsAuthenticated() {
		return authorization.UserID, nil
	}

	return primitive.NilObjectID, derp.Unauthorized("handler.profileUserID", "User is not authenticated")
}
