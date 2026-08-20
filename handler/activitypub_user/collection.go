package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetCollection serves one of a User's ActivityPub Collections, identified in the request URL
func GetCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, actorID *string, user *model.User) error {

	const location = "handler.activitypub_user.GetCollection"

	// Parse the CollectionID from the URL
	collectionID, err := primitive.ObjectIDFromHex(ctx.Param("collectionId"))

	if err != nil {
		return derp.Wrap(err, location, "Invalid collection ID in URL", "collectionId: "+ctx.Param("collectionId"), derp.WithNotFound())
	}

	// Load the collection
	collectionService := factory.Collection()
	record := model.NewCollection()

	if err := collectionService.LoadByID(session, user.UserID, collectionID, &record); err != nil {
		return derp.Wrap(err, location, "Loading collection", "userId", user.UserID.Hex(), "collectionId", collectionID.Hex())
	}

	// Compute URLs for the collection and collection owner
	userActivityPubURL := user.ActivityPubURL()
	collectionURL := userActivityPubURL + "/pub/collections/" + collectionID.Hex()

	// RULE: Only the owner, a domain owner, or a named participant may view this collection
	if !canViewCollection(ctx, &record, *actorID) {
		return derp.Forbidden(location, "Actor does not have permission to view this collection", "actor", *actorID, "collectionId", collectionID.Hex())
	}

	// Serve "the collection", which is the full reply chain that matches this collection url
	collectionItemService := factory.CollectionItem()

	return collection.Serve(ctx,
		collectionURL,
		collectionItemService.CollectionCount(session, user.UserID, collectionID, exp.All()),
		collectionItemService.CollectionIterator(session, user.UserID, collectionID, exp.All()),
		collection.WithAttributedTo(userActivityPubURL),
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
