package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/directory/
func GetDirectory(serverFactory *server.Factory) func(model.Authorization, txn.GetDirectory) ([]object.Account, toot.PageInfo, error) {

	return func(model.Authorization, txn.GetDirectory) ([]object.Account, toot.PageInfo, error) {
		return []object.Account{}, toot.PageInfo{}, nil
	}
}
