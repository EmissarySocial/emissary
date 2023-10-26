package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/preferences/
func GetPreferences(serverFactory *server.Factory) func(model.Authorization, txn.GetPreferences) (map[string]any, error) {

	return func(model.Authorization, txn.GetPreferences) (map[string]any, error) {

	}
}
