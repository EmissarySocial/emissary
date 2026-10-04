package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/preferences/
func GetPreferences(serverFactory *server.Factory) func(model.Authorization, txn.GetPreferences) (object.Preferences, error) {

	return func(model.Authorization, txn.GetPreferences) (object.Preferences, error) {

		result := object.Preferences{
			PostingDefaultVisibility: "public",
			PostingDefaultSensitive:  false,
			PostingDefaultLanguage:   "",
			ReadingExpandMedia:       "default",
			ReadingExpandSpoilers:    false,
		}

		return result, nil
	}
}
