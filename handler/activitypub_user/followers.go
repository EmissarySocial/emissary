package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/steranko"
)

// GetFollowersCollection publishes the SIZE of the User's followers collection, but never its members.
//
// RULE: Emissary does not enumerate follower identities to the network.  Publishing that list
// discloses data about the *followers*, who never consented to it -- an account owner can consent
// to publishing their own profile, but not on everyone else's behalf.  ActivityPub §5.3 permits
// this explicitly: the collection "MAY be filtered on privileges of an authenticated user or as
// appropriate when no authentication is given."
//
// collection.ServeSummary emits the count with no `first` page, which is the shape Mastodon and
// GoToSocial serve for an actor who has hidden their social graph -- and which Mastodon reads back
// as "this user chose to hide their follows" rather than "this user has no followers."
func GetFollowersCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetFollowersCollection"

	// RULE: A non-public User's follower count is not public information
	if !isUserVisible(ctx, user) {
		return derp.NotFound(location, "User not found")
	}

	// RULE: This collection is never enumerated, so paging into it is forbidden -- not empty.
	// Answering 403 (as Mastodon and Misskey do) keeps "you may not read this" distinct from
	// "there is nothing here," and prevents a future paging implementation from leaking by default.
	if isPagingRequest(ctx) {
		return derp.Forbidden(location, "This collection cannot be enumerated")
	}

	// The count is denormalized onto the User record, and deliberately includes every follower
	// method (ActivityPub and Email alike) -- it answers "how many people receive this actor's
	// posts," not "how many ActivityPub records exist."
	return collection.ServeSummary(ctx, user.ActivityPubFollowersURL(), int64(user.FollowerCount))
}
