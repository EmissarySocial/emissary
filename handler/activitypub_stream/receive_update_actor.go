package activitypub_stream

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/sherlock"
)

func init() {

	// An actor's profile Update routes here (exact routes beat any Update wildcard),
	// mirroring the Delete/Person handler in receive_delete_person.go.
	streamRouter.Add(vocab.ActivityTypeUpdate, vocab.ActorTypePerson, receive_UpdateActor)
	streamRouter.Add(vocab.ActivityTypeUpdate, vocab.ActorTypeApplication, receive_UpdateActor)
	streamRouter.Add(vocab.ActivityTypeUpdate, vocab.ActorTypeGroup, receive_UpdateActor)
	streamRouter.Add(vocab.ActivityTypeUpdate, vocab.ActorTypeOrganization, receive_UpdateActor)
	streamRouter.Add(vocab.ActivityTypeUpdate, vocab.ActorTypeService, receive_UpdateActor)
}

// receive_UpdateActor handles a remote actor's profile Update with cache invalidation +
// refetch, keeping the ActivityStream cache from serving a stale profile until TTL expiry.
func receive_UpdateActor(context Context, activity streams.Document) error {

	const location = "handler.activitypub_stream.receive_UpdateActor"

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

	// Fresh as a daisy
	return nil
}
