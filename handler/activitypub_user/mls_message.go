package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/steranko"
)

func GetMLSMessageCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	mlsMessageService := factory.MLSMessage()

	return collection.Serve(ctx,
		mlsMessageService.CollectionID(user.UserID),
		mlsMessageService.CollectionCount(session, user.UserID),
		mlsMessageService.CollectionIterator(session, user.UserID),
	)
}

func GetMLSMessageRecord(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetMLSMessageRecord"

	// Load the mlsMessage from the database
	mlsMessageService := factory.MLSMessage()
	mlsMessage := model.NewMLSMessage()

	if err := mlsMessageService.LoadByToken(session, user.UserID, ctx.Param("messageId"), &mlsMessage); err != nil {
		return derp.Wrap(err, location, "Unable to load MLS message")
	}

	// Return the MLS message to the client in JSON-LD format
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, mlsMessage.GetJSONLD())
}
