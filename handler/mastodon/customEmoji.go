package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/custom_emojis/
func GetCustomEmojis(serverFactory *server.Factory) func(model.Authorization, txn.GetCustomEmojis) ([]object.CustomEmoji, error) {

	return func(model.Authorization, txn.GetCustomEmojis) ([]object.CustomEmoji, error) {
		return []object.CustomEmoji{}, nil
	}
}
