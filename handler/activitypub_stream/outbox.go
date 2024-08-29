package activitypub_stream

import (
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
	"github.com/labstack/echo/v4"
)

func GetOutboxCollection(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "activitypub_stream.GetOutboxCollection"

	return func(ctx echo.Context) error {

		// Load all of the necessary object from the request
		factory, _, _, _, stream, actor, err := getActor(serverFactory, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Request Not Accepted")
		}

		if actor.IsNil() {
			return derp.NewNotFoundError(location, "Actor not found")
		}

		// If the request is for the collection itself, then return a summary and the URL of the first page
		publishDateString := ctx.QueryParam("publishDate")

		if publishDateString == "" {
			ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
			result := activitypub.Collection(stream.ActivityPubOutboxURL())
			return ctx.JSON(http.StatusOK, result)
		}

		// Fall through means that we're looking for a specific page of the collection
		publishedDate := convert.Int64Default(publishDateString, math.MaxInt64)
		pageSize := 60

		// Retrieve a page of messages from the database
		outboxService := factory.Outbox()
		messages, err := outboxService.QueryByParentAndDate(model.FollowerTypeStream, stream.StreamID, publishedDate, 60)

		if err != nil {
			return derp.Wrap(err, location, "Error loading outbox messages")
		}

		// Return results as an OrderedCollectionPage
		ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
		result := activitypub.CollectionPage(stream.ActivityPubOutboxURL(), pageSize, messages)
		return ctx.JSON(http.StatusOK, result)
	}
}
