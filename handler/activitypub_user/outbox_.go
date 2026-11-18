package activitypub_user

import (
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/assanitizer"
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

		totalItems, lastPageDate, err := outboxService.PagingSummary(session, model.ActorTypeUser, user.UserID, permissions, activitypub.OutboxPageSize)

		if err != nil {
			return derp.Wrap(err, location, "Counting outbox activities")
		}

		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		result := activitypub.PagedCollection(user.ActivityPubOutboxURL(), totalItems, lastPageDate)
		return ctx.JSON(http.StatusOK, result)
	}

	// Fall through means that we're looking for a specific page of the collection
	publishedDate := convert.Int64Default(publishDateString, math.MaxInt64)
	pageID := fullURL(factory, ctx)

	// Retrieve a page of activities from the database
	items, err := outboxService.QueryByParentAndDate(session, model.ActorTypeUser, user.UserID, permissions, publishedDate, activitypub.OutboxPageSize)

	if err != nil {
		return derp.Wrap(err, location, "Loading outbox activities")
	}

	// Return results as an OrderedCollectionPage
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	result := activitypub.CollectionPage(pageID, user.ActivityPubOutboxURL(), activitypub.OutboxPageSize, items)
	return ctx.JSON(http.StatusOK, result)
}

// GetOutboxActivity returns a specific activity from the User's outbox
func GetOutboxActivity(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetOutboxActivity"

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFound(location, "User not found")
	}

	// Get the OutboxMessageID from the context
	outboxMessageToken := ctx.Param("messageId")
	outboxMessageID, err := primitive.ObjectIDFromHex(outboxMessageToken)

	if err != nil {
		return derp.Wrap(err, location, "OutboxMessageID must be a valid ObjectID", outboxMessageToken, derp.WithNotFound())
	}

	// Load the activity from this User's outbox
	item := model.NewOutboxItem()

	if err := factory.Outbox2().LoadByID(session, model.ActorTypeUser, user.UserID, outboxMessageID, &item); err != nil {
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

	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, result)
}

// PostOutbox allows an Authenticated User to POST messages to their outbox, to be delivered to the
// network according to the message's delivery
func PostOutbox(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.PostOutbox"

	// Get ActivityStream client for this User
	client := factory.ActivityStream().UserClient(user.UserID)

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

	// Retrieve the activity from the request body
	activity, err := router.ReceiveRequest(ctx.Request(), client, matchActor)

	if err != nil {
		return derp.Wrap(err, location, "Receiving ActivityPub request")
	}

	// RULE: reserved "emissary:" properties are server-generated only; a client cannot speak in the
	// server's voice, so any inbound ones are stripped before the activity is stored or delivered.
	assanitizer.Strip(activity.Value(), model.NamespaceEmissary)

	// Route the activity to the correct handler
	if err := outboxRouter.Handle(context, activity); err != nil {
		return derp.Wrap(err, location, "Handling ActivityPub request")
	}

	// RULE: Most handlers write their own response. One that wrote nothing published nothing, so it answers 200.
	if ctx.Response().Committed {
		return nil
	}

	return ctx.NoContent(http.StatusOK)
}
