package activitypub_user

import (
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/router"
	"github.com/benpate/hannibal/validator"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetOutboxCollection returns the User's outbox as an ActivityPub OrderedCollection
func GetOutboxCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetOutboxCollection"

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFound(location, "User not found")
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

// GetOutboxActivity returns a specific activity from the User's outbox
func GetOutboxActivity(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetOutboxCollection"

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFound(location, "User not found")
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
	if err := outboxService.LoadByID(session, user.UserID, outboxMessageID, &outboxMessage); err != nil {
		return derp.Wrap(err, location, "Unable to load outbox message", outboxMessageID)
	}

	// Return results as an OrderedCollectionPage
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, outboxMessage.GetJSONLD())
}

// PostOutbox allows an Authenticated User to POST messages to their outbox, to be delivered to the
// network according to the message's delivery
func PostOutbox(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.PostOutbox"

	// Get ActivityStream service for this User
	activityService := factory.ActivityStream(model.ActorTypeUser, user.UserID)

	// Create a new Context
	context := Context{
		context: ctx,
		factory: factory,
		session: session,
		user:    user,
	}

	// Use custom validation because this route has already been validated,
	// but we want to guarantee that the actor in the Activity matches the
	// authenticated user.
	matchActor := router.WithValidators(
		validator.NewMatchActor(user.ActivityPubURL()),
	)

	// Retrieve the activity from the request body and route it to the correct handler
	if err := outboxRouter.ReceiveAndHandle(context, ctx.Request(), activityService.Client(), matchActor); err != nil {
		return derp.Wrap(err, location, "Unable to handle ActivityPub request")
	}

	// Handler writes its response directly to the context
	return ctx.NoContent(http.StatusOK)
}
