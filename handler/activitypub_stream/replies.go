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

// GetRepliesCollection serves the ActivityPub "replies" collection for a Stream,
// reading from the Stream's JIT Replies collection (Type "Replies", parent =
// this Stream). An empty collection is served when the Stream has no replies yet.
func GetRepliesCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, stream *model.Stream) error {

	const location = "handler.activitypub_stream.GetRepliesCollection"

	// RULE: Only PUBLIC streams expose their replies.
	if !stream.DefaultAllowAnonymous() {
		return derp.Unauthorized(location, "Anonymous access not allowed")
	}

	baseRequestURL := stream.ActivityPubRepliesURL()

	// If no "publishDate" query param, return the collection header pointing at the first page.
	publishDateString := ctx.QueryParam("publishDate")

	if publishDateString == "" {
		ctx.Response().Header().Set("Content-Type", model.MimeTypeActivityPub)
		return ctx.JSON(http.StatusOK, activitypub.Collection(baseRequestURL))
	}

	// Fall through means a specific page was requested. Locate the Replies collection.
	collectionService := factory.Collection()
	collection := model.NewCollection()

	if err := collectionService.LoadByParentAndType(session, stream.StreamID, model.CollectionTypeReplies, &collection); err != nil {

		// No Replies collection yet just means no replies. Serve an empty page.
		if derp.IsNotFound(err) {
			ctx.Response().Header().Set("Content-Type", model.MimeTypeActivityPub)
			return ctx.JSON(http.StatusOK, activitypub.CollectionPage_Links(fullURL(factory, ctx), baseRequestURL, 0, []model.CollectionItem{}))
		}

		return derp.Wrap(err, location, "Unable to load Replies collection")
	}

	// Retrieve a page of reply items, ordered by createDate.
	publishedDate := convert.Int64(publishDateString)
	pageSize := 60

	items, err := factory.CollectionItem().QueryByCollectionAndDate(session, collection.CollectionID, publishedDate, pageSize)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load reply items")
	}

	// Serve the page as a collection of links (each item is a reply URI).
	ctx.Response().Header().Set("Content-Type", model.MimeTypeActivityPub)
	return ctx.JSON(http.StatusOK, activitypub.CollectionPage_Links(fullURL(factory, ctx), baseRequestURL, pageSize, items))
}
