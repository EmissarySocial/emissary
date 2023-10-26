package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/oembed/
func GetOEmbed(serverFactory *server.Factory) func(model.Authorization, txn.GetOEmbed) (map[string]any, error) {

	return func(model.Authorization, txn.GetOEmbed) (map[string]any, error) {

	}
}
