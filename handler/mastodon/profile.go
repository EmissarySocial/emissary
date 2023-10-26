package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/profile/
func DeleteProfile_Avatar(serverFactory *server.Factory) func(model.Authorization, txn.DeleteProfile_Avatar) (object.Account, error) {

	return func(model.Authorization, txn.DeleteProfile_Avatar) (object.Account, error) {

	}
}

func DeleteProfile_Header(serverFactory *server.Factory) func(model.Authorization, txn.DeleteProfile_Header) (object.Account, error) {

	return func(model.Authorization, txn.DeleteProfile_Header) (object.Account, error) {

	}
}
