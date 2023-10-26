package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/emails/
func PostEmailConfirmation(serverFactory *server.Factory) func(model.Authorization, txn.PostEmailConfirmation) (struct{}, error) {

	return func(model.Authorization, txn.PostEmailConfirmation) (struct{}, error) {

	}
}
