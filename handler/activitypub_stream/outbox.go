package activitypub_stream

import (
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/steranko"
)

func GetOutboxCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream) error {

	const location = "handler.activitypub_stream.GetOutboxCollection"

	// Verify the stream is an ActivityPub actor
	if template.Actor.IsNil() {
		return derp.NotFoundError(location, "Actor not found")
	}

	permissions := factory.Permission().ParseHTTPSignature(session, ctx.Request())

	// If the request is for the collection itself, then return a summary and the URL of the first page
	publishDateString := ctx.QueryParam("publishDate")

	if publishDateString == "" {
		ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
		result := activitypub.Collection(stream.ActivityPubOutboxURL())
		return ctx.JSON(http.StatusOK, result)
	}

	// Fall through means that we're looking for a specific page of the collection
	publishedDate := convert.Int64Default(publishDateString, math.MaxInt64)
	pageID := fullURL(factory, ctx)
	pageSize := 60

	// Retrieve a page of messages from the database
	outboxService := factory.Outbox()
	messages, err := outboxService.QueryByParentAndDate(session, model.FollowerTypeStream, stream.StreamID, permissions, publishedDate, 60)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load outbox messages")
	}

	// Return results as an OrderedCollectionPage
	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	result := activitypub.CollectionPage(pageID, stream.ActivityPubOutboxURL(), pageSize, messages)
	return ctx.JSON(http.StatusOK, result)
}
