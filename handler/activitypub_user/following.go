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

// GetFollowingCollection publishes the SIZE of the User's following collection, but never its
// members.  See GetFollowersCollection for the policy and the reasoning behind the response shape.
func GetFollowingCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetFollowingCollection"

	// RULE: A non-public User's following count is not public information
	if !isUserVisible(ctx, user) {
		return derp.NotFound(location, "User not found")
	}

	// RULE: This collection is never enumerated, so paging into it is forbidden -- not empty.
	if isPagingRequest(ctx) {
		return derp.Forbidden(location, "This collection cannot be enumerated")
	}

	// As with followers, the count is inclusive of every method: RSS/Atom/JSONFeed subscriptions
	// (FollowingMethodPoll) count alongside ActivityPub follows, because the number answers
	// "how many sources does this actor read."
	return collection.ServeSummary(ctx, user.ActivityPubFollowingURL(), int64(user.FollowingCount))
}

// GetFollowingRecord serves a single Following record from the User's following collection
func GetFollowingRecord(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetFollowingRecord"

	// Confirm that the user is visible
	if !isUserVisible(ctx, user) {
		return ctx.NoContent(http.StatusNotFound)
	}

	// Load the following from the database
	followingService := factory.Following()
	following := model.NewFollowing()

	if err := followingService.LoadByToken(session, user.UserID, ctx.Param("followingId"), &following); err != nil {
		return derp.Wrap(err, location, "Loading following")
	}

	result := followingService.AsJSONLD(&following)

	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, result)
}
