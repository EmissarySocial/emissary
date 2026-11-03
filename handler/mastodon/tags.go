package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// PostTag_Follow implements the Mastodon "follow a hashtag" endpoint, which Emissary does not have (422)
func PostTag_Follow(serverFactory *server.Factory) func(model.Authorization, txn.PostTag_Follow) (object.Tag, error) {

	// RULE: Refused rather than faked, so the app does not show a hashtag as followed when nothing follows it
	return func(model.Authorization, txn.PostTag_Follow) (object.Tag, error) {
		return object.Tag{}, errNotSupported("handler.mastodon.PostTag_Follow", "Followed hashtags")
	}
}

// PostTag_Unfollow implements the Mastodon "unfollow a hashtag" endpoint, which Emissary does not have (422)
func PostTag_Unfollow(serverFactory *server.Factory) func(model.Authorization, txn.PostTag_Unfollow) (object.Tag, error) {

	return func(model.Authorization, txn.PostTag_Unfollow) (object.Tag, error) {
		return object.Tag{}, errNotSupported("handler.mastodon.PostTag_Unfollow", "Followed hashtags")
	}
}
