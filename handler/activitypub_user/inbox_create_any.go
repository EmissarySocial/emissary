package activitypub_user

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

// init registers the Create and Update handlers, and the object types that are deliberately ignored
func init() {

	// Wildcard to handle Create/Update of (nearlly) any type
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.Any, inbox_CreateOrUpdate)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.Any, inbox_CreateOrUpdate)

	// These values are skipped
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeRelationship, inbox_Unknown)
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeProfile, inbox_Unknown)
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeTombstone, inbox_Unknown)

	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeRelationship, inbox_Unknown)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeProfile, inbox_Unknown)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeTombstone, inbox_Unknown)
}

// inbox_CreateOrUpdate adds a public Create or Update activity to the User's newsfeed
func inbox_CreateOrUpdate(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_CreateOrUpdate"

	// RULE: No additional processing for non-public activites. These have already
	// been added to the User's inbox (in inbox_SaveActivity) so they'll be picked
	// up by the chat client without any further action.
	if activity.NotPublic() {
		return nil
	}

	// Locate the original "object" value as an actual object.  If the object is
	// embedded inline, use it directly.  If it is a bare URL, load it from the
	// Interwebs -- and treat a load failure as a (retryable) error, so a transient
	// network problem does not silently drop the news item.
	document := activity.UnwrapActivity()

	if document.IsString() {

		loaded, err := document.Load()

		if err != nil {
			return derp.Wrap(err, location, "Loading embedded object", document.Value())
		}

		document = loaded
	}

	// Place the post into the User's newsfeed IF it came from a source they Follow. A post from a
	// non-followed actor is legitimate — most replies to this User's posts arrive from actors the
	// User does not follow — so a missing Following record is NOT an error: we skip the newsfeed
	// placement and fall through to the context/reply bookkeeping below. Returning an error here
	// would fail the whole inbox POST (which runs in a transaction — see handler.WithFactory),
	// rolling back the Reply/Mention notification created centrally in PostInbox and triggering
	// endless sender retries. (Mirrors the accept-or-silently-drop pattern in inbox_LikeOrAnnounce.)
	followingService := context.factory.Following()
	following := model.NewFollowing()

	if err := followingService.LoadByURL(context.session, context.user.UserID, activity.ActorID(), &following); err == nil {

		// Followed source: save the post into the User's newsfeed (with de-duplication).
		if err := followingService.SaveNewsItem(context.session, &following, document, model.OriginTypePrimary); err != nil {
			return derp.Wrap(err, location, "Saving news item", context.user.UserID, activity.Value())
		}

	} else if derp.IsNotFound(err) {
		// Not a followed source: no newsfeed placement. Continue to the context bookkeeping below.

	} else {
		// A real load error (not merely "no record") — surface it.
		return derp.Wrap(err, location, "Loading `Following` record", context.user.UserID)
	}

	// Add this document to a context (if necessary)
	if hasLocalReplyOrContext(document, context.factory.Host()) {

		postcommit.Publish(
			context.session,
			context.factory.Queue(),
			"AddToCollection",
			mapof.Any{
				"hostname": context.factory.Hostname(), // The host that received the activity
				"userId":   context.user.UserID.Hex(),  // The user who received the activity
				"actorId":  activity.ActorID(),         // The actor adding the item to the context
				"url":      document.ID(),              // The URL to add to a context collection
			},
		)
	}

	// Success!
	return nil
}

// hasLocalReplyOrContext returns TRUE if the document belongs to a context that
// is owned by this server, or replies to a document that is owned by this server.
func hasLocalReplyOrContext(document streams.Document, host string) bool {

	if documentContext := document.Context(); strings.HasPrefix(documentContext, host) {
		return true
	}

	if inReplyTo := document.InReplyTo().ID(); strings.HasPrefix(inReplyTo, host) {
		return true
	}

	return false
}
