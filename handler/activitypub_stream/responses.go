package activitypub_stream

import (
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/steranko"
)

// GetLikesCollection serves the ActivityPub "likes" collection for a Stream.
func GetLikesCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, stream *model.Stream) error {
	return getResponseCollection(ctx, factory, session, stream, model.CollectionTypeLikes, stream.ActivityPubLikesURL())
}

// GetDislikesCollection serves the ActivityPub "dislikes" collection for a Stream.
func GetDislikesCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, stream *model.Stream) error {
	return getResponseCollection(ctx, factory, session, stream, model.CollectionTypeDislikes, stream.ActivityPubDislikesURL())
}

// GetSharesCollection serves the ActivityPub "shares" collection for a Stream.
func GetSharesCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, stream *model.Stream) error {
	return getResponseCollection(ctx, factory, session, stream, model.CollectionTypeShares, stream.ActivityPubSharesURL())
}

// getResponseCollection serves a Stream's Like/Dislike collection from its JIT
// CollectionItem projection. It serves an empty collection when there are none yet.
func getResponseCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, stream *model.Stream, collectionType string, baseRequestURL string) error {

	const location = "handler.activitypub_stream.getResponseCollection"

	// RULE: Only PUBLIC streams expose their response collections.
	if !stream.DefaultAllowAnonymous() {
		return derp.Unauthorized(location, "Anonymous access not allowed")
	}

	// Locate the response collection (needed for both the header's totalItems and the page items).
	collectionService := factory.Collection()
	collection := model.NewCollection()
	collectionExists := true

	if err := collectionService.LoadByType(session, stream.StreamID, collectionType, &collection); err != nil {

		// No collection yet just means no responses of this type (totalItems = 0).
		if derp.IsNotFound(err) {
			collectionExists = false
		} else {
			return derp.Wrap(err, location, "Loading response collection", collectionType)
		}
	}

	// If no "publishDate" query param, return the collection header (with totalItems) pointing at
	// the first page. totalItems is part of the W3C collection definition (D9).
	publishDateString := ctx.QueryParam("publishDate")

	if publishDateString == "" {
		result := activitypub.Collection(baseRequestURL)
		result.TotalItems = collection.TotalItems
		ctx.Response().Header().Set("Content-Type", model.MimeTypeActivityPub)
		return ctx.JSON(http.StatusOK, result)
	}

	// A specific page was requested but no collection exists yet: serve an empty page.
	if !collectionExists {
		ctx.Response().Header().Set("Content-Type", model.MimeTypeActivityPub)
		return ctx.JSON(http.StatusOK, activitypub.CollectionPage_Links(fullURL(factory, ctx), baseRequestURL, 0, []model.CollectionItem{}))
	}

	// Retrieve a page of response items, ordered by createDate.
	publishedDate := convert.Int64(publishDateString)
	pageSize := 60

	items, err := factory.CollectionItem().QueryByCollectionAndDate(session, collection.CollectionID, publishedDate, pageSize)

	if err != nil {
		return derp.Wrap(err, location, "Loading response items")
	}

	// Serve the page as a collection of links (each item is a response activity URI).
	ctx.Response().Header().Set("Content-Type", model.MimeTypeActivityPub)
	return ctx.JSON(http.StatusOK, activitypub.CollectionPage_Links(fullURL(factory, ctx), baseRequestURL, pageSize, items))
}
