package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/followed_tags/
func GetFollowedTags(serverFactory *server.Factory) func(model.Authorization, txn.GetFollowedTags) ([]object.Tag, toot.PageInfo, error) {

	return func(model.Authorization, txn.GetFollowedTags) ([]object.Tag, toot.PageInfo, error) {
		return []object.Tag{}, toot.PageInfo{}, nil
	}
}
