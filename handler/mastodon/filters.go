package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
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
		return object.Filter{}, derp.NewInternalError("handler.mastodon.PostFilter", "Not Implemented")
	}
}

func PutFilter(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter) (object.Filter, error) {

	return func(model.Authorization, txn.PutFilter) (object.Filter, error) {
		return object.Filter{}, derp.NewInternalError("handler.mastodon.PutFilter", "Not Implemented")
	}
}

func DeleteFilter(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter) (struct{}, error) {
		return struct{}{}, derp.NewInternalError("handler.mastodon.DeleteFilter", "Not Implemented")
	}
}

func GetFilter_Keywords(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Keywords) ([]string, error) {

	return func(model.Authorization, txn.GetFilter_Keywords) ([]string, error) {
		return []string{}, nil
	}
}

func PostFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_Keyword) (struct{}, error) {

	return func(model.Authorization, txn.PostFilter_Keyword) (struct{}, error) {
		return struct{}{}, derp.NewInternalError("handler.mastodon.PostFilter_Keyword", "Not Implemented")
	}
}

func GetFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Keyword) (object.FilterKeyword, error) {

	return func(model.Authorization, txn.GetFilter_Keyword) (object.FilterKeyword, error) {
		return object.FilterKeyword{}, nil
	}
}

func PutFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter_Keyword) (object.FilterKeyword, error) {

	return func(model.Authorization, txn.PutFilter_Keyword) (object.FilterKeyword, error) {
		return object.FilterKeyword{}, derp.NewInternalError("handler.mastodon.PutFilter_Keyword", "Not Implemented")
	}
}

func DeleteFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_Keyword) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_Keyword) (struct{}, error) {
		return struct{}{}, derp.NewInternalError("handler.mastodon.DeleteFilter_Keyword", "Not Implemented")
	}
}

func GetFilter_Statuses(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Statuses) ([]object.FilterStatus, error) {

	return func(model.Authorization, txn.GetFilter_Statuses) ([]object.FilterStatus, error) {
		return []object.FilterStatus{}, nil
	}
}

func PostFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_Status) (object.FilterStatus, error) {

	return func(model.Authorization, txn.PostFilter_Status) (object.FilterStatus, error) {
		return object.FilterStatus{}, derp.NewInternalError("handler.mastodon.PostFilter_Status", "Not Implemented")
	}
}

func GetFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Status) (object.FilterStatus, error) {

	return func(model.Authorization, txn.GetFilter_Status) (object.FilterStatus, error) {
		return object.FilterStatus{}, nil
	}
}

func DeleteFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_Status) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_Status) (struct{}, error) {
		return struct{}{}, derp.NewInternalError("handler.mastodon.DeleteFilter_Status", "Not Implemented")
	}
}

func GetFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.GetFilter_V1) (object.Filter, error) {
		return object.Filter{}, nil
	}
}

func PostFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.PostFilter_V1) (object.Filter, error) {
		return object.Filter{}, derp.NewInternalError("handler.mastodon.PostFilter_V1", "Not Implemented")
	}
}

func PutFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.PutFilter_V1) (object.Filter, error) {
		return object.Filter{}, derp.NewInternalError("handler.mastodon.PutFilter_V1", "Not Implemented")
	}
}

func DeleteFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_V1) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_V1) (struct{}, error) {
		return struct{}{}, derp.NewInternalError("handler.mastodon.DeleteFilter_V1", "Not Implemented")
	}
}
