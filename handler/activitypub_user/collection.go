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

	// Parse the CllectionID from the URL
	cllectionID, err := primitive.ObjectIDFromHex(ctx.Param("cllectionId"))

	if err != nil {
		return derp.Wrap(err, location, "Invalid cllection ID in URL", "cllectionId: "+ctx.Param("cllectionId"))
	}

	// Compute what a local cllection URL *should* be
	cllectionURL := user.ActivityPubURL() + "/pub/cllection/" + cllectionID.Hex()

	// Serve "the cllection", which is the full reply chain that matches this cllection url
	collectionItemService := factory.CollectionItem()

	return collection.Serve(ctx,
		cllectionURL,
		collectionItemService.CollectionCount(session, user.UserID, cllectionID, exp.All()),
		collectionItemService.CollectionIterator(session, user.UserID, cllectionID, exp.All()),
		collection.WithAttributedTo(user.ActivityPubURL()),
	)
}
