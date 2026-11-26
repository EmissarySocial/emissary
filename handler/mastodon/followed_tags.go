package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// followedTagsDefault is Mastodon's default number of followed tags per page
const followedTagsDefault = 100

// followedTagsMax is Mastodon's largest number of followed tags per page
const followedTagsMax = 200

// GetFollowedTags implements the Mastodon "followed hashtags" endpoint, newest first
// https://docs.joinmastodon.org/methods/followed_tags/
func GetFollowedTags(serverFactory *server.Factory) func(model.Authorization, txn.GetFollowedTags) ([]object.Tag, toot.PageInfo, error) {

	const location = "handler.mastodon.GetFollowedTags"

	return func(auth model.Authorization, t txn.GetFollowedTags) ([]object.Tag, toot.PageInfo, error) {

		// RULE: Followed tags are personal, so the caller must be signed in
		if !auth.IsAuthenticated() {
			return nil, toot.PageInfo{}, derp.Unauthorized(location, "Sign in to list followed hashtags")
		}

		// Get the Domain factory and a database session
		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return nil, toot.PageInfo{}, err
		}

		defer cancel()

		// Load the caller's followed tags
		follows, err := factory.TagFollow().QueryByUser(session, auth.UserID, followedTagsLimit(t.Limit))

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Querying followed tags")
		}

		// Describe each one as a Mastodon Tag
		result := make([]object.Tag, 0, len(follows))

		for _, follow := range follows {
			result = append(result, followedTag(t.Host, follow.Name, true))
		}

		// Your hashtags, sir
		return result, toot.PageInfo{}, nil
	}
}

// followedTagsLimit returns the number of followed tags to send, within Mastodon's bounds
func followedTagsLimit(limit int64) int64 {

	if limit <= 0 {
		return followedTagsDefault
	}

	return min(limit, followedTagsMax)
}
