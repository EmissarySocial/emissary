package activitypub_user

import (
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetOutboxCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetOutboxCollection"

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFoundError(location, "User not found")
	}

	// If the request is for the collection itself, then return a summary and the URL of the first page
	publishDateString := ctx.QueryParam("publishDate")

	if publishDateString == "" {
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		result := activitypub.Collection(user.ActivityPubOutboxURL())
		return ctx.JSON(http.StatusOK, result)
	}

	// Retrieve permissions from the request signature
	permissions := factory.Permission().ParseHTTPSignature(session, ctx.Request())

	// Fall through means that we're looking for a specific page of the collection
	publishedDate := convert.Int64Default(publishDateString, math.MaxInt64)
	outboxService := factory.Outbox()
	pageID := fullURL(factory, ctx)
	pageSize := 60

	// Retrieve a page of messages from the database
	messages, err := outboxService.QueryByParentAndDate(session, model.FollowerTypeUser, user.UserID, permissions, publishedDate, pageSize)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load outbox messages")
	}

	// Return results as an OrderedCollectionPage
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	result := activitypub.CollectionPage(pageID, user.ActivityPubOutboxURL(), pageSize, messages)
	return ctx.JSON(http.StatusOK, result)
}

func GetOutboxActivity(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetOutboxCollection"

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFoundError(location, "User not found")
	}

	// Get the OutboxMessageID from the context
	outboxMessageToken := ctx.Param("messageId")
	outboxMessageID, err := primitive.ObjectIDFromHex(outboxMessageToken)

	if err != nil {
		return derp.Wrap(err, location, "OutboxMessageID must be a valid ObjectID", outboxMessageToken)
	}

	// Load the Outbox Message
	outboxService := factory.Outbox()
	outboxMessage := model.NewOutboxMessage()
	if err := outboxService.LoadByID(session, outboxMessageID, &outboxMessage); err != nil {
		return derp.Wrap(err, location, "Unable to load outbox message", outboxMessageID)
	}

	// Return results as an OrderedCollectionPage
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, outboxMessage.GetJSONLD())
}
