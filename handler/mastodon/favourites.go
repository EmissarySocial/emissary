package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/favourites/
func GetFavourites(serverFactory *server.Factory) func(model.Authorization, txn.GetFavourites) ([]object.Status, error) {

	return func(model.Authorization, txn.GetFavourites) ([]object.Status, error) {
		return []object.Status{}, nil
	}
}
