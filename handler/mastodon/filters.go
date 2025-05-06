package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/filters/
func GetFilters(serverFactory *server.Factory) func(model.Authorization, txn.GetFilters) ([]object.Filter, error) {

	return func(model.Authorization, txn.GetFilters) ([]object.Filter, error) {
		return []object.Filter{}, nil
	}
}

func GetFilter(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter) (object.Filter, error) {

	return func(model.Authorization, txn.GetFilter) (object.Filter, error) {
		return object.Filter{}, nil
	}
}

func PostFilter(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter) (object.Filter, error) {

	return func(model.Authorization, txn.PostFilter) (object.Filter, error) {
		return object.Filter{}, derp.NotImplementedError("handler.mastodon.PostFilter")
	}
}

func PutFilter(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter) (object.Filter, error) {

	return func(model.Authorization, txn.PutFilter) (object.Filter, error) {
		return object.Filter{}, derp.NotImplementedError("handler.mastodon.PutFilter")
	}
}

func DeleteFilter(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter) (struct{}, error) {
		return struct{}{}, derp.NotImplementedError("handler.mastodon.DeleteFilter")
	}
}

func GetFilter_Keywords(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Keywords) ([]string, error) {

	return func(model.Authorization, txn.GetFilter_Keywords) ([]string, error) {
		return []string{}, nil
	}
}

func PostFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_Keyword) (struct{}, error) {

	return func(model.Authorization, txn.PostFilter_Keyword) (struct{}, error) {
		return struct{}{}, derp.NotImplementedError("handler.mastodon.PostFilter_Keyword")
	}
}

func GetFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Keyword) (object.FilterKeyword, error) {

	return func(model.Authorization, txn.GetFilter_Keyword) (object.FilterKeyword, error) {
		return object.FilterKeyword{}, nil
	}
}

func PutFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter_Keyword) (object.FilterKeyword, error) {

	return func(model.Authorization, txn.PutFilter_Keyword) (object.FilterKeyword, error) {
		return object.FilterKeyword{}, derp.NotImplementedError("handler.mastodon.PutFilter_Keyword")
	}
}

func DeleteFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_Keyword) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_Keyword) (struct{}, error) {
		return struct{}{}, derp.NotImplementedError("handler.mastodon.DeleteFilter_Keyword")
	}
}

func GetFilter_Statuses(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Statuses) ([]object.FilterStatus, error) {

	return func(model.Authorization, txn.GetFilter_Statuses) ([]object.FilterStatus, error) {
		return []object.FilterStatus{}, nil
	}
}

func PostFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_Status) (object.FilterStatus, error) {

	return func(model.Authorization, txn.PostFilter_Status) (object.FilterStatus, error) {
		return object.FilterStatus{}, derp.NotImplementedError("handler.mastodon.PostFilter_Status")
	}
}

func GetFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Status) (object.FilterStatus, error) {

	return func(model.Authorization, txn.GetFilter_Status) (object.FilterStatus, error) {
		return object.FilterStatus{}, nil
	}
}

func DeleteFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_Status) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_Status) (struct{}, error) {
		return struct{}{}, derp.NotImplementedError("handler.mastodon.DeleteFilter_Status")
	}
}

func GetFilters_V1(serverFactory *server.Factory) func(model.Authorization, txn.GetFilters_V1) ([]object.Filter, toot.PageInfo, error) {

	return func(model.Authorization, txn.GetFilters_V1) ([]object.Filter, toot.PageInfo, error) {
		return []object.Filter{}, toot.PageInfo{}, nil
	}
}

func GetFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.GetFilter_V1) (object.Filter, error) {
		return object.Filter{}, nil
	}
}

func PostFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.PostFilter_V1) (object.Filter, error) {
		return object.Filter{}, derp.NotImplementedError("handler.mastodon.PostFilter_V1")
	}
}

func PutFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.PutFilter_V1) (object.Filter, error) {
		return object.Filter{}, derp.NotImplementedError("handler.mastodon.PutFilter_V1")
	}
}

func DeleteFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_V1) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_V1) (struct{}, error) {
		return struct{}{}, derp.NotImplementedError("handler.mastodon.DeleteFilter_V1")
	}
}
