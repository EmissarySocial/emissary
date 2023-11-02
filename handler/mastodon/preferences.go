package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/preferences/
func GetPreferences(serverFactory *server.Factory) func(model.Authorization, txn.GetPreferences) (map[string]any, error) {

	return func(model.Authorization, txn.GetPreferences) (map[string]any, error) {

		result := map[string]any{
			"posting:default:visibility": "public",
			"posting:default:sensitive":  false,
			"posting:default:language":   nil,
			"reading:expand:media":       "default",
			"reading:expand:spoilers":    false,
		}

		return result, nil
	}
}
