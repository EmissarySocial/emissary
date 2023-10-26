package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/tags/
func GetTag(serverFactory *server.Factory) func(model.Authorization, txn.GetTag) (object.Tag, error) {

	return func(model.Authorization, txn.GetTag) (object.Tag, error) {

	}
}

func PostTag_Follow(serverFactory *server.Factory) func(model.Authorization, txn.PostTag_Follow) (object.Tag, error) {

	return func(model.Authorization, txn.PostTag_Follow) (object.Tag, error) {

	}
}

func PostTag_Unfollow(serverFactory *server.Factory) func(model.Authorization, txn.PostTag_Unfollow) (object.Tag, error) {

	return func(model.Authorization, txn.PostTag_Unfollow) (object.Tag, error) {

	}
}
