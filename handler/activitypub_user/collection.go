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

func GetCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetCollection"

	// Parse the CollectionID from the URL
	collectionID, err := primitive.ObjectIDFromHex(ctx.Param("collectionId"))

	if err != nil {
		return derp.Wrap(err, location, "Invalid collection ID in URL", "collectionId: "+ctx.Param("collectionId"))
	}

	// Compute what a local collection URL *should* be
	collectionURL := user.ActivityPubURL() + "/pub/collection/" + collectionID.Hex()

	// Serve "the collection", which is the full reply chain that matches this collection url
	collectionItemService := factory.CollectionItem()

	return collection.Serve(ctx,
		collectionURL,
		collectionItemService.CollectionCount(session, user.UserID, collectionID, exp.All()),
		collectionItemService.CollectionIterator(session, user.UserID, collectionID, exp.All()),
		collection.WithAttributedTo(user.ActivityPubURL()),
	)
}
