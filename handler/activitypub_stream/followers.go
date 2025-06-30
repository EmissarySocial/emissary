package activitypub_stream

import (
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/labstack/echo/v4"
)

func GetFollowersCollection(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.activitypub_stream.GetFollowersCollection"

	return func(ctx echo.Context) error {

		factory, _, _, _, stream, actor, err := getActor(serverFactory, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error getting actor")
		}

		if actor.IsNil() {
			return derp.NotFoundError(location, "Actor not found")
		}

		// If the request is for the collection itself, then return a summary and the URL of the first page
		publishDateString := ctx.QueryParam("publishDate")

		if publishDateString == "" {
			ctx.Response().Header().Set("Content-Type", "application/activity+json")
			result := activitypub.Collection(stream.ActivityPubFollowersURL())
			return ctx.JSON(http.StatusOK, result)
		}

		// Fall through means that we're looking for a specific page of the collection
		publishedDate := convert.Int64Default(publishDateString, math.MaxInt64)
		pageID := fullURL(factory, ctx)
		pageSize := 60

		// Retrieve a page of messages from the database
		followerService := factory.Follower()
		followers, err := followerService.QueryByParentAndDate(model.FollowerTypeStream, stream.StreamID, model.FollowerMethodActivityPub, publishedDate, pageSize)

		if err != nil {
			return derp.Wrap(err, location, "Error querying followers")
		}

		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		result := activitypub.CollectionPage_Links(pageID, stream.ActivityPubFollowersURL(), pageSize, followers)
		return ctx.JSON(http.StatusOK, result)
	}
}
