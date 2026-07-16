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
	activityService := factory.ActivityStream()
	client := activityService.UserClient(user.UserID)

	// Receive the activity from the request, verifying HTTP signatures using our
	// own PublicKeyFinder (which looks up the key by the signature's keyID and
	// bypasses the cache, to avoid verifying against a stale signing key).
	activity, err := router.ReceiveRequest(
		ctx.Request(),
		client,

		// Injecting our own key finder that is aware of the ascache middleware.
		router.WithPublicKeyFinder(activityService.PublicKeyFinder),
	)

	if err != nil {
		return derp.Wrap(err, location, "Receiving ActivityPub request")
	}

	// Prevent duplicate actiities from being processes multiple times (e.g. due to retries or multiple deliveries)
	if inbox_IsDuplicateActivity(context, activity) {
		return nil
	}

	// Validate the Activity meets basic criteria to be processed.
	if err := inbox_ValidateActivity(activity); err != nil {
		return derp.Wrap(err, location, "Validating ActivityPub request", activity.Value())
	}

	// Save the activity to the actor's Inbox
	if err := inbox_SaveActivity(context, activity); err != nil {
		return derp.Wrap(err, location, "Saving activity to inbox", activity.Value())
	}

	// Create Notifications for this activity (mentions, replies, reactions).  This runs centrally,
	// regardless of Following state, because the per-type router handlers below intentionally drop
	// exactly the cases notifications care about.  A notification failure must NOT fail the inbox
	// request, so we report-and-continue.  FOLLOW notifications are the exception — see
	// inbox_follow_any.go (they fire after the Accept is sent).
	if err := context.factory.Notification().NotifyFromActivity(context.session, context.user, activity); err != nil {
		derp.Report(derp.Wrap(err, location, "Creating notifications for activity", activity.ID()))
	}

	// Route the activity to additional handlers to process side effects
	if err := inboxRouter.Handle(context, activity); err != nil {
		return derp.Wrap(err, location, "Handling ActivityPub request")
	}

	// Send the response to the client
	return ctx.String(http.StatusOK, "")
}

// inbox_IsDuplicateActivity checks if this activity has already been received and processed in the inbox
func inbox_IsDuplicateActivity(context Context, activity streams.Document) bool {
	return context.factory.Inbox().IsDuplicateActivity(context.session, context.user.UserID, activity.ID())
}

// inbox_ValidateActivity performs additional validate on activities received in the inbox.
// This is called before routing the activity to the appropriate handler, so it applies to all activities
func inbox_ValidateActivity(activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_ValidateActivity"

	// Require that the Activity has a valid ActorID
	if actorID := activity.ActorID(); actorID == "" {
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

	const location = "handler.activitypub_user.inbox_SaveActivity"

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
	inboxActivity.ActorID = activity.ActorID()
	inboxActivity.ActivityID = activity.ID()
	inboxActivity.Context = activity.Context()
	inboxActivity.ActivityType = activity.Type()
	inboxActivity.ObjectType = object.Type()
	inboxActivity.ObjectID = object.ID()
	inboxActivity.MediaType = object.MediaType()
	inboxActivity.ReceivedDate = time.Now().UnixMilli()
	inboxActivity.RawActivity = activity.Map()
	inboxActivity.IsPublic = activity.IsPublic()

	// PublishedDate is stored in MILLISECONDS (see model.InboxActivity and the outbox2 write path),
	// so use UnixMilli — .Unix() here would store seconds and sort this activity ~1000x too early.
	if publishedDate := activity.Published(); !publishedDate.IsZero() {
		inboxActivity.PublishedDate = publishedDate.UnixMilli()
	} else {
		inboxActivity.PublishedDate = time.Now().UnixMilli()
	}

	// Save the Activity to the User's Inbox
	if err := inboxService.Save(context.session, &inboxActivity, ""); err != nil {
		return derp.Wrap(err, location, "Saving direct message", context.user.UserID, activity.Value())
	}

	// Suxxess
	return nil
}
