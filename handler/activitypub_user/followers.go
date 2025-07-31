package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/steranko"
)

func GetFollowersCollection(ctx *steranko.Context, factory *domain.Factory, session data.Session) error {
	collectionID := fullURL(factory, ctx)
	result := streams.NewOrderedCollection(collectionID)
	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	return ctx.JSON(http.StatusOK, result)
}
