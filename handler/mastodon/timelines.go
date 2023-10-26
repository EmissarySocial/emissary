package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/timelines/
func GetTimeline_Public(serverFactory *server.Factory) func(model.Authorization, txn.GetTimeline_Public) ([]object.Status, error) {

	return func(model.Authorization, txn.GetTimeline_Public) ([]object.Status, error) {

	}
}

func GetTimeline_Hashtag(serverFactory *server.Factory) func(model.Authorization, txn.GetTimeline_Hashtag) ([]object.Status, error) {

	return func(model.Authorization, txn.GetTimeline_Hashtag) ([]object.Status, error) {

	}
}

func GetTimeline_Home(serverFactory *server.Factory) func(model.Authorization, txn.GetTimeline_Home) ([]object.Status, error) {

	return func(model.Authorization, txn.GetTimeline_Home) ([]object.Status, error) {

	}
}

func GetTimeline_List(serverFactory *server.Factory) func(model.Authorization, txn.GetTimeline_List) ([]object.Status, error) {

	return func(model.Authorization, txn.GetTimeline_List) ([]object.Status, error) {

	}
}
