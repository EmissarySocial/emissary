package activitypub_stream

import (
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/steranko"
)

func GetFollowersCollection(ctx *steranko.Context, factory *domain.Factory, session data.Session, template *model.Template, stream *model.Stream) error {

	const location = "handler.activitypub_stream.GetFollowersCollection"

	// Verify the stream is an ActivityPub actor
	actor := template.Actor

	if actor.IsNil() {
		return derp.NotFoundError(location, "Actor not found")
	}

	// If the request is for the collection itself, then return a summary and the URL of the first page
	publishDateString := ctx.QueryParam("publishDate")

	if publishDateString == "" {
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		result := activitypub.Collection(stream.ActivityPubFollowersURL())
		return ctx.JSON(http.StatusOK, result)
	}

	// Fall through means that we're looking for a specific page of the collection
	publishedDate := convert.Int64Default(publishDateString, math.MaxInt64)
	pageID := fullURL(factory, ctx)
	pageSize := 60

	// Retrieve a page of messages from the database
	followerService := factory.Follower()
	followers, err := followerService.QueryByParentAndDate(session, model.FollowerTypeStream, stream.StreamID, model.FollowerMethodActivityPub, publishedDate, pageSize)

	if err != nil {
		return derp.Wrap(err, location, "Error querying followers")
	}

	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	result := activitypub.CollectionPage_Links(pageID, stream.ActivityPubFollowersURL(), pageSize, followers)
	return ctx.JSON(http.StatusOK, result)
}
