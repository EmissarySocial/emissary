package activitypub_user

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/sherlock"
)

// init registers the profile-Update handlers, one per Actor type
func init() {

	// An actor's profile Update routes here instead of the (Update, Any) news-item wildcard:
	// exact activity/object routes match before wildcards in the hannibal router.
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ActorTypePerson, inbox_UpdateActor)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ActorTypeApplication, inbox_UpdateActor)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ActorTypeGroup, inbox_UpdateActor)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ActorTypeOrganization, inbox_UpdateActor)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ActorTypeService, inbox_UpdateActor)
}

// inbox_UpdateActor handles a remote actor's profile Update with cache invalidation + refetch.
// Without it, the ActivityStream cache serves the stale profile until its TTL expires -- up to
// a month for actor documents (see tools/ascacherules).
func inbox_UpdateActor(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_UpdateActor"

	objectID := activity.Object().ID()

	// RULE: Actors can only update themselves, not other actors
	if activity.ActorID() != objectID {
		return derp.Forbidden(location, "Actor and Object must be the same", activity.ActorID(), objectID)
	}

	// Purge the stale actor from the ActivityStream cache
	if err := context.factory.ActivityStream().Delete(objectID); err != nil {
		return derp.Wrap(err, location, "Deleting cached actor", objectID)
	}

	// Refetch from the origin to re-warm the cache. The embedded object is deliberately NOT
	// trusted here; the origin document is authoritative. A refetch failure is reported but
	// not returned: the purge already guarantees correctness (the next read refetches), and
	// failing the inbox POST would make the sender retry a delivery we have fully processed.
	if _, err := context.factory.ActivityStream().AppClient().Load(objectID, sherlock.AsActor()); err != nil {
		derp.Report(derp.Wrap(err, location, "Refreshing actor", objectID))
	}

	// Lookin' good, bacon man!
	return nil
}
