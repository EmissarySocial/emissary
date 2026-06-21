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

func GetCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, actorID *string) error {

	const location = "handler.activitypub_user.GetCollection"

	// Parse the UserID from the URL
	userID, err := primitive.ObjectIDFromHex(ctx.Param("userId"))

	if err != nil {
		return derp.Wrap(err, location, "Invalid user ID in URL", "userId: "+ctx.Param("userId"))
	}

	// Parse the CollectionID from the URL
	collectionID, err := primitive.ObjectIDFromHex(ctx.Param("collectionId"))

	if err != nil {
		return derp.Wrap(err, location, "Invalid collection ID in URL", "collectionId: "+ctx.Param("collectionId"))
	}

	// Load the collection
	collectionService := factory.Collection()
	record := model.NewCollection()

	if err := collectionService.LoadByID(session, userID, collectionID, &record); err != nil {
		return derp.Wrap(err, location, "Loading collection", "userId", userID.Hex(), "collectionId", collectionID.Hex())
	}

	// Compute URLs for the collection and collection owner
	userService := factory.User()
	userActivityPubURL := userService.ActivityPubURL(userID)
	collectionURL := userActivityPubURL + "/pub/collections/" + collectionID.Hex()

	// RULE: Only the owner, a domain owner, or a named participant may view this collection
	if !canViewCollection(ctx, &record, *actorID, userActivityPubURL) {
		return derp.Forbidden(location, "Actor does not have permission to view this collection", "actor", *actorID, "collectionId", collectionID.Hex())
	}

	// Serve "the collection", which is the full reply chain that matches this collection url
	collectionItemService := factory.CollectionItem()

	return collection.Serve(ctx,
		collectionURL,
		collectionItemService.CollectionCount(session, userID, collectionID, exp.All()),
		collectionItemService.CollectionIterator(session, userID, collectionID, exp.All()),
		collection.WithAttributedTo(userActivityPubURL),
	)
}

// canViewCollection returns TRUE if the requesting actor may read the given Collection: the
// owner and domain owners always can, otherwise the actor must be a named participant (or the
// Collection must be public). ownerActorURL is the ActivityPub URL of the Collection's owner.
func canViewCollection(ctx *steranko.Context, collection *model.Collection, actorID string, ownerActorURL string) bool {

	// The Collection's owner can always read it, even if not named in its participant lists
	if actorID != "" && actorID == ownerActorURL {
		return true
	}

	// Domain owners can read everything
	if getAuthorization(ctx).DomainOwner {
		return true
	}

	// Otherwise, the actor must be a participant (or the Collection must be public)
	return collection.HasParticipant(actorID)
}
