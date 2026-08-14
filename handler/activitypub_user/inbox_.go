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

	// Items carry the "emissary:labels" property (4C): persisted receive-time stamps merged with the
	// viewer's current rules, so the client can hide or annotate without its own rule engine.
	inboxService := factory.Inbox()
	criteria := exp.Equal("isPublic", false)

	return collection.Serve(ctx,
		user.ActivityPubURL()+"/pub/inbox/direct-messages",
		inboxService.CollectionCount(session, user.UserID, criteria),
		inboxService.CollectionIteratorWithLabels(session, user.UserID, criteria, factory.Rule()),
		collection.WithSSEEndpoint(user.ActivityPubSSEEndpoint_Inbox_DirectMessages()),
	)
}

// GetInboxCollection_DirectMessages_MLS serves the MLS-encrypted direct messages in the User's inbox.
func GetInboxCollection_DirectMessages_MLS(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	// Items carry the "emissary:labels" property (4C) -- essential here, because MLS from
	// blocked/muted senders is stored rather than dropped and the labels are how the client knows.
	inboxService := factory.Inbox()
	criteria := exp.Equal("isPublic", false).AndEqual("mediaType", vocab.MediaTypeMLS)

	return collection.Serve(ctx,
		user.ActivityPubURL()+"/pub/inbox/direct-messages/mls",
		inboxService.CollectionCount(session, user.UserID, criteria),
		inboxService.CollectionIteratorWithLabels(session, user.UserID, criteria, factory.Rule()),
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
	// signature verification, then the reserved-namespace sanitizer. The verifier is a required
	// parameter of the funnel (BUG-19), so every inbox verifies signatures against keys loaded
	// through Emissary's client stack -- cache-aware, rules-gated, and bound by the private-IP policy.
	activity, err := activitypub.ReceiveRequest(
		ctx.Request(),
		client,
		activityService,
		factory.Rule(),
		session,
		user.UserID,
	)

	if err != nil {
		return derp.Wrap(err, location, "Receiving ActivityPub request")
	}

	// Drop duplicate activities so retries and multiple deliveries are processed only once
	if inbox_IsDuplicateActivity(context, activity) {
		return nil
	}

	// Validate the Activity meets basic criteria to be processed. This is the authoritative Stage-2
	// rule gate: it computes the sender's disposition ONCE and returns it for the steps below.
	disposition, err := inbox_ValidateActivity(context, activity)

	if err != nil {
		return derp.Wrap(err, location, "Validating ActivityPub request", activity.Value())
	}

	// RULE: A muted actor's plain (non-MLS) direct message is accepted but never stored (4B):
	// invisible to the sender, gone for the viewer, and idempotent on redelivery (no row, no dedup
	// hit). Returning here also skips notifications and routing, which follow storage.
	if inbox_SuppressStorage(disposition, activity) {
		return ctx.String(http.StatusOK, "")
	}

	// Save the activity to the actor's Inbox, stamped with the sender's disposition
	if err := inbox_SaveActivity(context, activity, disposition); err != nil {
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

// inbox_ValidateActivity performs additional validation on activities received in the inbox, and
// returns the sender's disposition.
func inbox_ValidateActivity(context Context, activity streams.Document) (model.RuleDisposition, error) {

	const location = "handler.activitypub_user.inbox_ValidateActivity"

	// This runs after HTTP-signature verification and before the activity is saved or routed, so it is
	// the authoritative Stage-2 rule gate (D5/D17) on the VERIFIED actor. The disposition is computed
	// here ONCE and threaded forward through the rest of the pipeline.

	// Require that the Activity has a valid ActorID
	if actorID := activity.ActorID(); actorID == "" {
		return model.RuleDisposition{}, derp.BadRequest(location, "Activity must have an ActorID", activity.Value())
	}

	// Require that the activity has a valid Type
	if activityType := activity.Type(); activityType == "" {
		return model.RuleDisposition{}, derp.BadRequest(location, "Activity must have a Type", activity.Value())
	}

	// RULE: An activity's id must share its actor's origin (D18). This closes the dedup-poisoning
	// primitive where an attacker pre-registers a victim's future activity id: the poisoned activity
	// is now rejected before it can be stored under that id. Only the top-level id is bound here;
	// cross-origin object references are legitimate and are bound separately (D19). A missing id
	// cannot poison (inbox_SaveActivity mints a local one), so it is exempt.
	if activityID := activity.ID(); activityID != "" {
		if !activitypub.IsSameOrigin(activity.ActorID(), activityID) {
			return model.RuleDisposition{}, derp.Unauthorized(location, "Activity id must share the actor's origin", activity.ActorID(), activityID)
		}
	}

	// Compute the sender's disposition ONCE, for every type -- exceptions included -- so the gate
	// below, storage stamping, and the served labels all read the same answer (4B/4C).
	disposition, err := context.factory.Rule().ActorDisposition(context.session, context.user.UserID, activity, time.Now().Unix())

	// Stage 2 fails CLOSED (D17): a rules-query failure must not let blocked content through.
	if err != nil {
		return model.RuleDisposition{}, derp.Wrap(err, location, "Checking inbound rules against the verified actor", activity.ActorID())
	}

	// RULE: Blocks stop inbound content at the front door (R1) -- by WHO is talking (ACTOR/DOMAIN),
	// never by content, and MUTE never gates the wire (D5). Exception-set types are verified but
	// handed to their per-type handler, never discarded here. Inline non-public MLS is NEVER dropped
	// (4B): it is accepted carrying this disposition, because a skipped ciphertext breaks the
	// conversation's epoch ratchet. The 401 is byte-and-status identical to a signature failure (D3)
	// so a blocked server cannot tell it was blocked (vs. a transient auth error).
	if disposition.IsBlocked() && !activitypub.IsWireGateException(activity.Type()) && !activitypub.IsMLSCreate(activity) {
		return model.RuleDisposition{}, derp.Unauthorized(location, "Cannot validate received activity", activity.ActorID())
	}

	// All good so far...
	return disposition, nil
}

// inbox_SuppressStorage returns TRUE for activities that are accepted but never stored: a muted
// actor's plain (non-MLS) direct message.
func inbox_SuppressStorage(disposition model.RuleDisposition, activity streams.Document) bool {

	// The Create-only scope below is load-bearing: Likes and Undos usually carry no addressing at all
	// (so IsPublic is FALSE for them), and suppressing those would break muted-actor aggregates (R9)
	// and subtractive actions (D6).

	// Only MUTE suppresses storage: blocked non-MLS content never gets this far, and clean
	// content always stores
	if !disposition.IsMuted() {
		return false
	}

	// Only a Create (an actual message delivery) is suppressed
	if activity.Type() != vocab.ActivityTypeCreate {
		return false
	}

	// Public posts store as today: the newsfeed walk (3F) already keeps them out of view
	if activity.IsPublic() {
		return false
	}

	// MLS is never dropped (4B)
	if activitypub.IsMLSCreate(activity) {
		return false
	}

	// Muted + plain + private: accepted, never stored. Poof.
	return true
}

// inbox_SaveActivity saves a received activity into the target User's inbox, stamped with the
// sender's disposition.
func inbox_SaveActivity(context Context, activity streams.Document, disposition model.RuleDisposition) error {

	const location = "handler.activitypub_user.inbox_SaveActivity"

	// The stamp is what makes stored rows self-describing (4C).

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
	inboxActivity.Disposition = disposition

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
