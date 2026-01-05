package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/steranko"
)

func GetMLSInbox(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	mlsInboxService := factory.MLSInbox()

	return collection.Serve(ctx,
		mlsInboxService.CollectionID(user.UserID),
		mlsInboxService.CollectionCount(session, user.UserID),
		mlsInboxService.CollectionIterator(session, user.UserID),
	)

}
