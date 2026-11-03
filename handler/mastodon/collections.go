package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// GetAccount_Collections answers an account's featured collections with an empty list
// https://docs.joinmastodon.org/methods/collections/
func GetAccount_Collections(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Collections) ([]object.Collection, toot.PageInfo, error) {

	// Emissary has no featured collections, and an empty list lets clients draw the profile

	return func(model.Authorization, txn.GetAccount_Collections) ([]object.Collection, toot.PageInfo, error) {
		return []object.Collection{}, toot.PageInfo{}, nil
	}
}
