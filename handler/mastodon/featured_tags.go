package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/featured_tags/
func GetFeaturedTags(serverFactory *server.Factory) func(model.Authorization, txn.GetFeaturedTags) ([]object.FeaturedTag, error) {

	return func(model.Authorization, txn.GetFeaturedTags) ([]object.FeaturedTag, error) {
		return []object.FeaturedTag{}, nil
	}
}

func PostFeaturedTag(serverFactory *server.Factory) func(model.Authorization, txn.PostFeaturedTag) (object.FeaturedTag, error) {

	return func(model.Authorization, txn.PostFeaturedTag) (object.FeaturedTag, error) {
		return object.FeaturedTag{}, derp.NewInternalError("handler.mastodon.PostFeaturedTag", "Not Implemented")
	}
}

func DeleteFeaturedTag(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFeaturedTag) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFeaturedTag) (struct{}, error) {
		return struct{}{}, derp.NewInternalError("handler.mastodon.PostFeaturedTag", "Not Implemented")
	}
}

func GetFeaturedTags_Suggestions(serverFactory *server.Factory) func(model.Authorization, txn.GetFeaturedTags_Suggestions) ([]object.FeaturedTag, error) {

	return func(model.Authorization, txn.GetFeaturedTags_Suggestions) ([]object.FeaturedTag, error) {
		return []object.FeaturedTag{}, nil
	}
}
