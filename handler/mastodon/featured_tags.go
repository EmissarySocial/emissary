package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/featured_tags/
func GetFeaturedTags(serverFactory *server.Factory) func(model.Authorization, txn.GetFeaturedTags) ([]object.FeaturedTag, error) {

	return func(model.Authorization, txn.GetFeaturedTags) ([]object.FeaturedTag, error) {

	}
}

func PostFeaturedTag(serverFactory *server.Factory) func(model.Authorization, txn.PostFeaturedTag) (object.FeaturedTag, error) {

	return func(model.Authorization, txn.PostFeaturedTag) (object.FeaturedTag, error) {

	}
}

func DeleteFeaturedTag(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFeaturedTag) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFeaturedTag) (struct{}, error) {

	}
}

func GetFeaturedTags_Suggestions(serverFactory *server.Factory) func(model.Authorization, txn.GetFeaturedTags_Suggestions) ([]object.FeaturedTag, error) {

	return func(model.Authorization, txn.GetFeaturedTags_Suggestions) ([]object.FeaturedTag, error) {

	}
}
