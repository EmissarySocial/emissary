package activitypub_stream

import (
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/slice"
	"github.com/labstack/echo/v4"
)

func GetOutboxCollection(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "activitypub_stream.GetOutboxCollection"

	return func(ctx echo.Context) error {

		// Load all of the necessary object from the request
		_, _, streamService, _, stream, _, err := getActor(serverFactory, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Invalid Request")
		}

		// If the request is for the collection itself, then return a summary and the URL of the first page
		publishDateString := ctx.QueryParam("publishDate")

		if publishDateString == "" {
			ctx.Response().Header().Set("Content-Type", "application/activity+json")
			result := activitypub.Collection(stream.ActivityPubOutboxURL())
			return ctx.JSON(http.StatusOK, result)
		}

		// Fall through means that we're looking for a specific page of the collection
		publishedDate := convert.Int64Default(publishDateString, math.MaxInt64)
		pageSize := 60

		// Retrieve a page of messages from the database
		streams, err := streamService.QueryByParentAndDate(stream.StreamID, publishedDate, pageSize)

		if err != nil {
			return derp.Wrap(err, location, "Error loading outbox messages")
		}

		getters := slice.Map(streams, func(stream model.Stream) service.StreamJSONLDGetter {
			return streamService.JSONLDGetter(&stream)
		})

		// Return results as an OrderedCollectionPage
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		result := activitypub.CollectionPage(stream.ActivityPubOutboxURL(), pageSize, getters)
		return ctx.JSON(http.StatusOK, result)
	}
}
