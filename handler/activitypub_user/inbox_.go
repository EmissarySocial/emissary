package activitypub_user

import (
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/hannibal/router"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetInboxCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	inboxService := factory.Inbox()

	return collection.Serve(ctx,
		user.ActivityPubURL()+"/pub/inbox",
		inboxService.CollectionCount(session, user.UserID, exp.All()),
		inboxService.CollectionIterator(session, user.UserID, exp.All()),
		collection.WithSSEEndpoint(user.ActivityPubSSEEndpoint_Inbox()),
	)
}

func GetInboxCollection_DirectMessages(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	inboxService := factory.Inbox()
	criteria := exp.Equal("isPublic", false)

	return collection.Serve(ctx,
		user.ActivityPubURL()+"/pub/inbox/direct-messages",
		inboxService.CollectionCount(session, user.UserID, criteria),
		inboxService.CollectionIterator(session, user.UserID, criteria),
		collection.WithSSEEndpoint(user.ActivityPubSSEEndpoint_Inbox_DirectMessages()),
	)
}

func GetInboxCollection_DirectMessages_MLS(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	inboxService := factory.Inbox()
	criteria := exp.Equal("isPublic", false).AndEqual("mediaType", vocab.MediaTypeMLS)

	return collection.Serve(ctx,
		user.ActivityPubURL()+"/pub/inbox/direct-messages/mls",
		inboxService.CollectionCount(session, user.UserID, criteria),
		inboxService.CollectionIterator(session, user.UserID, criteria),
		collection.WithSSEEndpoint(user.ActivityPubSSEEndpoint_Inbox_DirectMessages_MLS()),
	)
}

func PostInbox(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.PostInbox"

	// Create a new Context
	context := Context{
		context: ctx,
		factory: factory,
		session: session,
		user:    user,
	}

	// Get ActivityStream service for this User
	client := factory.ActivityStream().UserClient(user.UserID)

	// Receive the activity from the request (with optional options)
	activity, err := router.ReceiveRequest(ctx.Request(), client)

	if err != nil {
		return derp.Wrap(err, location, "Unable to receive ActivityPub request")
	}

	// Validate the Activity meets basic criteria to be processed.
	if err := inbox_ValidateActivity(activity); err != nil {
		return derp.Wrap(err, location, "Unable to validate ActivityPub request", activity.Value())
	}

	// Save the activity to the actor's Inbox
	if err := inbox_SaveActivity(context, activity); err != nil {
		return derp.Wrap(err, location, "Unable to save activity to inbox", activity.Value())
	}

	// Route the activity to additional handlers to process side effects
	if err := inboxRouter.Handle(context, activity); err != nil {
		return derp.Wrap(err, location, "Unable to handle ActivityPub request")
	}

	// Send the response to the client
	return ctx.String(http.StatusOK, "")
}

// inbox_ValidateActivity performs additional validate on activities received in the inbox.
// This is called before routing the activity to the appropriate handler, so it applies to all activities
func inbox_ValidateActivity(activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_ValidateActivity"

	// Require that the Activity has a valid ActorID
	if actorID := activity.Actor().ID(); actorID == "" {
		return derp.BadRequest(location, "Activity must have an ActorID", activity.Value())
	}

	// Require that the activity has a valid Type
	if activityType := activity.Type(); activityType == "" {
		return derp.BadRequest(location, "Activity must have a Type", activity.Value())
	}

	// ADDITIONAL VALIDATION LOGIC GOES HERE...
	// Rules/Blocks

	// All good so far...
	return nil
}

// inbox_SaveActivity accepts all activities that are delivered to this actor, and
// saves them into their inbox
func inbox_SaveActivity(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_DirectMessage"

	/* Require that the activity is addressed to this Actor
	if activity.Recipients().NotContains(context.user.ActivityPubURL()) {
		return derp.BadRequest(location, "Direct messages must be addressed to this Actor")
	} */

	// RULE: Create a default id for the activity if none is provided
	if activity.ID() == "" {
		activity.SetID("uri:uuid:" + primitive.NewObjectID().Hex())
	}

	// If not already a "map" then load the link to the object
	object := activity.Object().LoadLink()

	// Create a new InboxActivity and save it to the Inbox
	inboxService := context.factory.Inbox()
	inboxActivity := model.NewInboxActivity()
	inboxActivity.UserID = context.user.UserID
	inboxActivity.ActorID = activity.Actor().ID()
	inboxActivity.ActivityID = activity.ID()
	inboxActivity.Context = activity.Context()
	inboxActivity.ActivityType = activity.Type()
	inboxActivity.ObjectType = object.Type()
	inboxActivity.ObjectID = object.ID()
	inboxActivity.MediaType = object.MediaType()
	inboxActivity.ReceivedDate = time.Now().UnixMilli()
	inboxActivity.RawActivity = activity.Map()
	inboxActivity.IsPublic = activity.IsPublic()

	if publishedDate := activity.Published(); !publishedDate.IsZero() {
		inboxActivity.PublishedDate = publishedDate.Unix()
	} else {
		inboxActivity.PublishedDate = time.Now().Unix()
	}

	if err := inboxService.Save(context.session, &inboxActivity, ""); err != nil {
		return derp.Wrap(err, location, "Unable to save direct message", context.user.UserID, activity.Value())
	}

	return nil
}
