package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

func GetObjectsCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetObjectsCollection"

	return derp.NotImplemented(location, "Not Implemented")
}

func GetObject(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetObject"

	// Collect parameters and services
	objectService := factory.Object()
	object := model.NewObject()
	token := ctx.Param("objectId")

	// Retrieve the Object from the database
	if err := objectService.LoadByToken(session, user.UserID, token, &object); err != nil {
		return derp.Wrap(err, location, "Unable to load object", "token", token)
	}

	// RULE: The requester must be allowed to view this Object (public, or a named recipient)
	if !canViewObjectPermissions(ctx, factory, object.Permissions) {
		return derp.Forbidden(location, "Access denied")
	}

	// Return the object value as JSON
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, object.Value)
}
