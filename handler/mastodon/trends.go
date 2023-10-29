package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/trends/
func GetTrends(serverFactory *server.Factory) func(model.Authorization, txn.GetTrends) ([]object.Tag, error) {

	return func(model.Authorization, txn.GetTrends) ([]object.Tag, error) {
		return []object.Tag{}, nil
	}
}

func GetTrends_Statuses(serverFactory *server.Factory) func(model.Authorization, txn.GetTrends_Statuses) ([]object.Status, error) {

	return func(model.Authorization, txn.GetTrends_Statuses) ([]object.Status, error) {
		return []object.Status{}, nil
	}
}

func GetTrends_Links(serverFactory *server.Factory) func(model.Authorization, txn.GetTrends_Links) ([]object.PreviewCard, error) {

	return func(model.Authorization, txn.GetTrends_Links) ([]object.PreviewCard, error) {
		return []object.PreviewCard{}, nil
	}
}
