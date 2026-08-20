package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/steranko"
)

// serveCollection serves one of a Stream's ActivityPub collections, enforcing its access permissions first
func serveCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, actorID *string, stream *model.Stream, collectionType string, collectionURL string) error {

	const location = "handler.activitypub_stream.serveCollection"

	// Load the Collection (to enforce access permissions)
	collectionService := factory.Collection()
	record, err := collectionService.LoadOrCreateByStream(session, stream, collectionType)

	if err != nil {

		if derp.IsNotFound(err) {
			return collection.ServeEmpty(ctx, collectionURL)
		}

		return derp.Wrap(err, location, "Loading collection")
	}

	// RULE: Enforce access permissions on the collection
	if !canViewCollection(ctx, &record, *actorID) {
		return derp.Forbidden(location, "Collection not readable by "+*actorID)
	}

	collectionItemService := factory.CollectionItem()

	return collection.Serve(
		ctx,
		collectionURL,
		collectionItemService.CollectionCount(session, stream.StreamID, record.CollectionID, exp.All()),
		collectionItemService.CollectionIterator(session, stream.StreamID, record.CollectionID, exp.All()),
	)
}

// canViewCollection returns TRUE if the requesting actor may read the given Collection: the
// owner and domain owners always can, otherwise the actor must be a named participant (or the
// Collection must be public). ownerActorURL is the ActivityPub URL of the Collection's owner.
func canViewCollection(ctx *steranko.Context, collection *model.Collection, actorID string) bool {

	authorization := getAuthorization(ctx)

	// Domain owners can read everything
	if authorization.DomainOwner {
		return true
	}

	// Collection owners can always read their own collections
	if authorization.UserID == collection.UserID {
		return true
	}

	// Otherwise, the actor must be a participant (or the Collection must be public)
	return collection.IsReadable(actorID)
}
