package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/endorsements/
func GetEndorsements(serverFactory *server.Factory) func(model.Authorization, txn.GetEndorsements) ([]object.Account, error) {

	return func(model.Authorization, txn.GetEndorsements) ([]object.Account, error) {
		return []object.Account{}, nil
	}
}
