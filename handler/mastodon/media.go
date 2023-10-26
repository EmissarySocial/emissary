package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/media/
func PostMedia(serverFactory *server.Factory) func(model.Authorization, txn.PostMedia) (object.MediaAttachment, error) {

	return func(model.Authorization, txn.PostMedia) (object.MediaAttachment, error) {

	}
}

// https://docs.joinmastodon.org/methods/mutes/
func GetMutes(serverFactory *server.Factory) func(model.Authorization, txn.GetMutes) ([]object.Account, error) {

	return func(model.Authorization, txn.GetMutes) ([]object.Account, error) {

	}
}
