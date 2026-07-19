package activitypub_user

import (
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/handler/activitypub"
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

// GetInboxCollection serves the User's ActivityPub inbox as a paged collection.
func GetInboxCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	inboxService := factory.Inbox()

	return collection.Serve(ctx,
		user.ActivityPubURL()+"/pub/inbox",
		inboxService.CollectionCount(session, user.UserID, exp.All()),
		inboxService.CollectionIterator(session, user.UserID, exp.All()),
		collection.WithSSEEndpoint(user.ActivityPubSSEEndpoint_Inbox()),
	)
}

// GetInboxCollection_DirectMessages serves the private (direct-message) subset of the User's inbox.
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

// GetInboxCollection_DirectMessages_MLS serves the MLS-encrypted direct messages in the User's inbox.
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

// PostInbox receives an inbound ActivityPub activity: it verifies the HTTP signature, drops
// duplicates, applies the Stage-2 rule gate, then saves, notifies, and routes the activity.
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

	// Receive and parse the activity through the canonical inbox receive funnel: Stage-1 validators,
	// signature verification, then the reserved-namespace sanitizer. The funnel puts the validator
	// chain before caller options, so our cache-aware key finder (which patches the HTTPSig entry in
	// place, so a stale cached signing key is never trusted) is never discarded.
	activity, err := activitypub.ReceiveRequest(
		ctx.Request(),
		client,
		factory.Rule(),
		session,
		user.UserID,

		// Injecting our own key finder that is aware of the ascache middleware.
		router.WithPublicKeyFinder(activityService.PublicKeyFinder),
	)

	if err != nil {
		return derp.Wrap(err, location, "Receiving ActivityPub request")
	}

	// Drop duplicate activities so retries and multiple deliveries are processed only once
	if inbox_IsDuplicateActivity(context, activity) {
		return nil
	}

	// Validate the Activity meets basic criteria to be processed.
	if err := inbox_ValidateActivity(context, activity); err != nil {
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

// inbox_ValidateActivity performs additional validation on activities received in the inbox. It runs
// after HTTP-signature verification and before the activity is saved or routed, so it is the
// authoritative Stage-2 rule gate (D5/D17) on the VERIFIED actor.
func inbox_ValidateActivity(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_ValidateActivity"

	// Require that the Activity has a valid ActorID
	if actorID := activity.ActorID(); actorID == "" {
		return derp.BadRequest(location, "Activity must have an ActorID", activity.Value())
	}

	// Require that the activity has a valid Type
	if activityType := activity.Type(); activityType == "" {
		return derp.BadRequest(location, "Activity must have a Type", activity.Value())
	}

	// RULE: An activity's id must share its actor's origin (D18). This closes the dedup-poisoning
	// primitive where an attacker pre-registers a victim's future activity id: the poisoned activity
	// is now rejected before it can be stored under that id. Only the top-level id is bound here;
	// cross-origin object references are legitimate and are bound separately (D19). A missing id
	// cannot poison (inbox_SaveActivity mints a local one), so it is exempt.
	if activityID := activity.ID(); activityID != "" {
		if !activitypub.IsSameOrigin(activity.ActorID(), activityID) {
			return derp.Unauthorized(location, "Activity id must share the actor's origin", activity.ActorID(), activityID)
		}
	}

	// RULE: Blocks stop inbound content at the front door (R1). The check is on the VERIFIED actor and
	// is authoritative. The gate blocks by WHO is talking (ACTOR/DOMAIN), never by content (TAG, which
	// is filtered at newsfeed ingest); and MUTE never gates the wire (D5) -- only BLOCK. Exception-set
	// types are verified but handed to their per-type handler, never discarded here.
	if !activitypub.IsWireGateException(activity.Type()) {

		blocked, err := context.factory.Rule().IsActorBlocked(context.session, context.user.UserID, activity)

		// Stage 2 fails CLOSED (D17): a rules-query failure must not let blocked content through.
		if err != nil {
			return derp.Wrap(err, location, "Checking inbound rules against the verified actor", activity.ActorID())
		}

		// A blocked actor's activity is discarded. The 401 is byte-and-status identical to a signature
		// failure (D3) so a blocked server cannot tell it was blocked (vs. a transient auth error).
		if blocked {
			return derp.Unauthorized(location, "Cannot validate received activity", activity.ActorID())
		}
	}

	// All good so far...
	return nil
}

// inbox_SaveActivity saves a received activity into the target User's inbox.
func inbox_SaveActivity(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_SaveActivity"

	// RULE: Create a default id for the activity if none is provided
	if activity.ID() == "" {
		activity.SetID("uri:uuid:" + primitive.NewObjectID().Hex())
	}

	// If not already a "map" then load the link to the object
	object := activity.Object().LoadLink()

	// Build the InboxActivity record from the received activity
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
