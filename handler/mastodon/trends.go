package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/trends/
func GetTrends(serverFactory *server.Factory) func(model.Authorization, txn.GetTrends) ([]object.Tag, toot.PageInfo, error) {

	return func(model.Authorization, txn.GetTrends) ([]object.Tag, toot.PageInfo, error) {
		return []object.Tag{}, toot.PageInfo{}, nil
	}
}

func GetTrends_Statuses(serverFactory *server.Factory) func(model.Authorization, txn.GetTrends_Statuses) ([]object.Status, toot.PageInfo, error) {

	return func(model.Authorization, txn.GetTrends_Statuses) ([]object.Status, toot.PageInfo, error) {
		return []object.Status{}, toot.PageInfo{}, nil
	}
}

func GetTrends_Links(serverFactory *server.Factory) func(model.Authorization, txn.GetTrends_Links) ([]object.PreviewCard, toot.PageInfo, error) {

	return func(model.Authorization, txn.GetTrends_Links) ([]object.PreviewCard, toot.PageInfo, error) {
		return []object.PreviewCard{}, toot.PageInfo{}, nil
	}
}
