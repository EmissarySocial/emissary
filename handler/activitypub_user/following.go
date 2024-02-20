package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

func GetFollowingCollection(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		result := streams.NewOrderedCollection()
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		return ctx.JSON(http.StatusOK, result)
	}
}

func GetFollowingRecord(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		sterankoContext, ok := ctx.(*steranko.Context)

		if !ok {
			return derp.NewInternalError("emissary.handler.ActivityPub_GetFollowingRecord", "Invalid context")
		}

		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "emissary.handler.ActivityPub_GetFollowingRecord", "Error loading server")
		}

		// Load the user from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByToken(ctx.Param("userId"), &user); err != nil {
			return derp.Wrap(err, "emissary.handler.ActivityPub_GetFollowingRecord", "Error loading user")
		}

		// Confirm that the user is visible
		if !isUserVisible(sterankoContext, &user) {
			return ctx.NoContent(http.StatusNotFound)
		}

		// Load the following from the database
		followingService := factory.Following()
		following := model.NewFollowing()

		if err := followingService.LoadByToken(user.UserID, ctx.Param("followingId"), &following); err != nil {
			return derp.Wrap(err, "emissary.handler.ActivityPub_GetFollowingRecord", "Error loading following")
		}

		result := followingService.AsJSONLD(&following)

		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		return ctx.JSON(http.StatusOK, result)
	}
}
