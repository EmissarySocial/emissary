package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetObjectsCollection serves a User's Objects collection
func GetObjectsCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetObjectsCollection"

	return derp.NotImplemented(location, "Not Implemented")
}

// GetObject serves a single Object owned by a User
func GetObject(ctx *steranko.Context, factory *service.Factory, session data.Session, actorID *string) error {

	const location = "handler.activitypub_user.GetObject"

	// Collect parameters and services
	objectService := factory.Object()
	object := model.NewObject()
	token := ctx.Param("objectId")

	// Parse the UserID from the query string
	userToken := ctx.Param("userId")

	userID, err := primitive.ObjectIDFromHex(userToken)

	if err != nil {
		return derp.Wrap(err, location, "Invalid UserID", userToken)
	}

	// Load the user record from the database
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByID(session, userID, &user); err != nil {
		return derp.Wrap(err, location, "Loading User", userID)
	}

	// Retrieve the Object from the database
	if err := objectService.LoadByToken(session, user.UserID, token, &object); err != nil {
		return derp.Wrap(err, location, "Loading Object", "token", token)
	}

	// RULE: The requester must be allowed to view this Object (public, or a named recipient)
	if objectService.NotAllowed(&object, *actorID) {
		return derp.Forbidden(location, "Access denied")
	}

	// Return the object value as JSON
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, object.Value)
}
