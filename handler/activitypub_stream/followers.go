package activitypub_stream

import (
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
)

// GetFollowersCollection serves the Followers collection for a Stream actor
func GetFollowersCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream) error {

	const location = "handler.activitypub_stream.GetFollowersCollection"

	// Verify permissions by checking the required permissions (stream.DefaultAllow) against the permissions in the request signature
	permissionService := factory.Permission()
	permissions := permissionService.ParseHTTPSignature(session, ctx.Request()) // nolint:scopeguard

	if !slice.ContainsAny(stream.DefaultAllow, permissions...) {
		return derp.Forbidden(location, "You do not have permission to view this content")
	}

	// Verify the stream is an ActivityPub actor
	if template.Actor.IsNil() {
		return derp.NotFound(location, "Actor not found")
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
		return derp.Wrap(err, location, "Querying followers")
	}

	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	result := activitypub.CollectionPage_Links(pageID, stream.ActivityPubFollowersURL(), pageSize, followers)
	return ctx.JSON(http.StatusOK, result)
}
