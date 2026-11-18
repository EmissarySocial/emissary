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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetOutboxCollection serves the Outbox collection for a Stream actor
func GetOutboxCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream) error {

	const location = "handler.activitypub_stream.GetOutboxCollection"

	// Verify the stream is an ActivityPub actor
	if template.Actor.IsNil() {
		return derp.NotFound(location, "Actor not found")
	}

	// Retrieve permissions from the request signature, which decide what the count and every page show
	permissions, err := factory.Permission().ParseHTTPSignature(session, ctx.Request())

	if err != nil {
		return derp.Wrap(err, location, "Invalid HTTP Signature")
	}

	// Read the page cursor, which is empty when the collection itself is requested
	outboxService := factory.Outbox2()
	publishDateString := ctx.QueryParam("publishDate")

	// If the request is for the collection itself, then return its size and the URLs of its first and last pages
	if publishDateString == "" {

		totalItems, lastPageDate, err := outboxService.PagingSummary(session, model.ActorTypeStream, stream.StreamID, permissions, activitypub.OutboxPageSize)

		if err != nil {
			return derp.Wrap(err, location, "Counting outbox activities")
		}

		ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
		result := activitypub.PagedCollection(stream.ActivityPubOutboxURL(), totalItems, lastPageDate)
		return ctx.JSON(http.StatusOK, result)
	}

	// Fall through means that we're looking for a specific page of the collection
	publishedDate := convert.Int64Default(publishDateString, math.MaxInt64)
	pageID := fullURL(factory, ctx)

	// Retrieve a page of activities from the database
	items, err := outboxService.QueryByParentAndDate(session, model.ActorTypeStream, stream.StreamID, permissions, publishedDate, activitypub.OutboxPageSize)

	if err != nil {
		return derp.Wrap(err, location, "Loading outbox activities")
	}

	// Return results as an OrderedCollectionPage
	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	result := activitypub.CollectionPage(pageID, stream.ActivityPubOutboxURL(), activitypub.OutboxPageSize, items)
	return ctx.JSON(http.StatusOK, result)
}

// GetOutboxMessage serves a single activity from a Stream actor's outbox
func GetOutboxMessage(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream) error {

	const location = "handler.activitypub_stream.GetOutboxMessage"

	// Verify the stream is an ActivityPub actor
	if template.Actor.IsNil() {
		return derp.NotFound(location, "Actor not found")
	}

	// Collect the OutboxMessageID from the URL path
	outboxMessageToken := ctx.Param("messageId")
	outboxMessageID, err := primitive.ObjectIDFromHex(outboxMessageToken)

	if err != nil {
		return derp.Wrap(err, location, "OutboxMessageID must be a valid ObjectID", outboxMessageToken, derp.WithNotFound())
	}

	// Load the activity, scoped to THIS Stream so one actor cannot serve another's activity
	item := model.NewOutboxItem()

	if err := factory.Outbox2().LoadByID(session, model.ActorTypeStream, stream.StreamID, outboxMessageID, &item); err != nil {
		return derp.Wrap(err, location, "Loading outbox activity", "activityId: "+outboxMessageToken)
	}

	return getOutboxItem(ctx, factory, session, &item)
}

// getOutboxItem writes an Outbox2 activity to the response, when the request's signer may read it
func getOutboxItem(ctx *steranko.Context, factory *service.Factory, session data.Session, item *model.OutboxItem) error {

	result, err := activitypub.VisibleOutboxItem(factory.Permission(), session, ctx.Request(), item)

	// RULE: Return a refusal as it is, so the peer reads its fixed message
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	return ctx.JSON(http.StatusOK, result)
}
