package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/steranko"
)

func GetFollowingCollection(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
	collectionID := fullURL(factory, ctx)
	result := streams.NewOrderedCollection(collectionID)
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, result)
}

func GetFollowingRecord(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.activitypub_user.GetFollowingRecord"

	// Load the user from the database
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByToken(session, ctx.Param("userId"), &user); err != nil {
		return derp.Wrap(err, location, "Error loading user")
	}

	// Confirm that the user is visible
	if !isUserVisible(ctx, &user) {
		return ctx.NoContent(http.StatusNotFound)
	}

	// Load the following from the database
	followingService := factory.Following()
	following := model.NewFollowing()

	if err := followingService.LoadByToken(session, user.UserID, ctx.Param("followingId"), &following); err != nil {
		return derp.Wrap(err, location, "Error loading following")
	}

	result := followingService.AsJSONLD(&following)

	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, result)
}
