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

// GetFilter implements the Mastodon "get filter" endpoint, and always returns an empty Filter
func GetFilter(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter) (object.Filter, error) {

	return func(model.Authorization, txn.GetFilter) (object.Filter, error) {
		return object.Filter{}, nil
	}
}

// PostFilter is the Mastodon "create filter" endpoint, which Emissary does not implement
func PostFilter(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter) (object.Filter, error) {

	return func(model.Authorization, txn.PostFilter) (object.Filter, error) {
		return object.Filter{}, derp.NotImplemented("handler.mastodon.PostFilter")
	}
}

// PutFilter is the Mastodon "update filter" endpoint, which Emissary does not implement
func PutFilter(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter) (object.Filter, error) {

	return func(model.Authorization, txn.PutFilter) (object.Filter, error) {
		return object.Filter{}, derp.NotImplemented("handler.mastodon.PutFilter")
	}
}

// DeleteFilter is the Mastodon "delete filter" endpoint, which Emissary does not implement
func DeleteFilter(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter) (struct{}, error) {
		return struct{}{}, derp.NotImplemented("handler.mastodon.DeleteFilter")
	}
}

// GetFilter_Keywords implements the Mastodon "get filter keywords" endpoint, and always returns an empty list
func GetFilter_Keywords(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Keywords) ([]string, error) {

	return func(model.Authorization, txn.GetFilter_Keywords) ([]string, error) {
		return []string{}, nil
	}
}

// PostFilter_Keyword is the Mastodon "add filter keyword" endpoint, which Emissary does not implement
func PostFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_Keyword) (struct{}, error) {

	return func(model.Authorization, txn.PostFilter_Keyword) (struct{}, error) {
		return struct{}{}, derp.NotImplemented("handler.mastodon.PostFilter_Keyword")
	}
}

// GetFilter_Keyword implements the Mastodon "get filter keyword" endpoint, and always returns an empty keyword
func GetFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Keyword) (object.FilterKeyword, error) {

	return func(model.Authorization, txn.GetFilter_Keyword) (object.FilterKeyword, error) {
		return object.FilterKeyword{}, nil
	}
}

// PutFilter_Keyword is the Mastodon "update filter keyword" endpoint, which Emissary does not implement
func PutFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter_Keyword) (object.FilterKeyword, error) {

	return func(model.Authorization, txn.PutFilter_Keyword) (object.FilterKeyword, error) {
		return object.FilterKeyword{}, derp.NotImplemented("handler.mastodon.PutFilter_Keyword")
	}
}

// DeleteFilter_Keyword is the Mastodon "delete filter keyword" endpoint, which Emissary does not implement
func DeleteFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_Keyword) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_Keyword) (struct{}, error) {
		return struct{}{}, derp.NotImplemented("handler.mastodon.DeleteFilter_Keyword")
	}
}

// GetFilter_Statuses implements the Mastodon "get filter statuses" endpoint, and always returns an empty list
func GetFilter_Statuses(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Statuses) ([]object.FilterStatus, error) {

	return func(model.Authorization, txn.GetFilter_Statuses) ([]object.FilterStatus, error) {
		return []object.FilterStatus{}, nil
	}
}

// PostFilter_Status is the Mastodon "add filter status" endpoint, which Emissary does not implement
func PostFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_Status) (object.FilterStatus, error) {

	return func(model.Authorization, txn.PostFilter_Status) (object.FilterStatus, error) {
		return object.FilterStatus{}, derp.NotImplemented("handler.mastodon.PostFilter_Status")
	}
}

// GetFilter_Status implements the Mastodon "get filter status" endpoint, and always returns an empty status
func GetFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Status) (object.FilterStatus, error) {

	return func(model.Authorization, txn.GetFilter_Status) (object.FilterStatus, error) {
		return object.FilterStatus{}, nil
	}
}

// DeleteFilter_Status is the Mastodon "delete filter status" endpoint, which Emissary does not implement
func DeleteFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_Status) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_Status) (struct{}, error) {
		return struct{}{}, derp.NotImplemented("handler.mastodon.DeleteFilter_Status")
	}
}

// GetFilters_V1 implements the deprecated V1 "get filters" endpoint, and always returns an empty list
func GetFilters_V1(serverFactory *server.Factory) func(model.Authorization, txn.GetFilters_V1) ([]object.Filter, toot.PageInfo, error) {

	return func(model.Authorization, txn.GetFilters_V1) ([]object.Filter, toot.PageInfo, error) {
		return []object.Filter{}, toot.PageInfo{}, nil
	}
}

// GetFilter_V1 implements the deprecated V1 "get filter" endpoint, and always returns an empty Filter
func GetFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.GetFilter_V1) (object.Filter, error) {
		return object.Filter{}, nil
	}
}

// PostFilter_V1 is the deprecated V1 "create filter" endpoint, which Emissary does not implement
func PostFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.PostFilter_V1) (object.Filter, error) {
		return object.Filter{}, derp.NotImplemented("handler.mastodon.PostFilter_V1")
	}
}

// PutFilter_V1 is the deprecated V1 "update filter" endpoint, which Emissary does not implement
func PutFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.PutFilter_V1) (object.Filter, error) {
		return object.Filter{}, derp.NotImplemented("handler.mastodon.PutFilter_V1")
	}
}

// DeleteFilter_V1 is the deprecated V1 "delete filter" endpoint, which Emissary does not implement
func DeleteFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_V1) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_V1) (struct{}, error) {
		return struct{}{}, derp.NotImplemented("handler.mastodon.DeleteFilter_V1")
	}
}
