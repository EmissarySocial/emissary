package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/suggestions/
func GetSuggestions(serverFactory *server.Factory) func(model.Authorization, txn.GetSuggestions) ([]object.Suggestion, error) {

	return func(model.Authorization, txn.GetSuggestions) ([]object.Suggestion, error) {
		return []object.Suggestion{}, nil
	}
}

func DeleteSuggestion(serverFactory *server.Factory) func(model.Authorization, txn.DeleteSuggestion) (struct{}, error) {

	return func(model.Authorization, txn.DeleteSuggestion) (struct{}, error) {
		return struct{}{}, nil
	}
}
