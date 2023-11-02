package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/search/
func GetSearch(serverFactory *server.Factory) func(model.Authorization, txn.GetSearch) (object.Search, error) {

	return func(model.Authorization, txn.GetSearch) (object.Search, error) {
		return object.Search{}, nil
	}
}
