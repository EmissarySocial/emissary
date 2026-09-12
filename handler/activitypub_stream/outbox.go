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
	"github.com/benpate/rosetta/slice"
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
		return derp.Wrap(err, location, "Loading outbox messages")
	}

	// Return results as an OrderedCollectionPage
	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	result := activitypub.CollectionPage(pageID, stream.ActivityPubOutboxURL(), pageSize, messages)
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

	// Load the OutboxMessage, scoped to THIS Stream so one actor cannot serve another's activity
	outboxService := factory.Outbox()
	outboxMessage := model.NewOutboxMessage()

	if err := outboxService.LoadByID(session, stream.StreamID, outboxMessageID, &outboxMessage); err != nil {
		return derp.Wrap(err, location, "Loading outbox message", outboxMessageID)
	}

	// RULE: The permissions in the HTTP signature must satisfy the message's own permissions.
	// The collection applies the same test as a query filter; a single item must apply it here.
	permissions := factory.Permission().ParseHTTPSignature(session, ctx.Request())

	if !slice.ContainsAny(outboxMessage.Permissions, permissions...) {
		return derp.Forbidden(location, "You do not have permission to view this content")
	}

	// RULE: A message with no Actor has no `id` to answer with, so there is nothing here to serve
	if outboxMessage.ActivityPubURL() == "" {
		return derp.NotFound(location, "Outbox message cannot be identified", outboxMessageID)
	}

	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	return ctx.JSON(http.StatusOK, outboxMessage.GetJSONLD())
}
