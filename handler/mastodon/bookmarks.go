package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/bookmarks/
func GetBookmarks(serverFactory *server.Factory) func(model.Authorization, txn.GetBookmarks) ([]object.Status, error) {

	// const location = "handler.mastodon_GetBookmarks"

	return func(auth model.Authorization, t txn.GetBookmarks) ([]object.Status, error) {
		return []object.Status{}, nil
	}
}
