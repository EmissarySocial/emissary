package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/filters/
func GetFilters(serverFactory *server.Factory) func(model.Authorization, txn.GetFilters) ([]object.Filter, error) {

	return func(model.Authorization, txn.GetFilters) ([]object.Filter, error) {

	}
}

func GetFilter(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter) (object.Filter, error) {

	return func(model.Authorization, txn.GetFilter) (object.Filter, error) {

	}
}

func PostFilter(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter) (object.Filter, error) {

	return func(model.Authorization, txn.PostFilter) (object.Filter, error) {

	}
}

func PutFilter(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter) (object.Filter, error) {

	return func(model.Authorization, txn.PutFilter) (object.Filter, error) {

	}
}

func DeleteFilter(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter) (struct{}, error) {

	}
}

func GetFilter_Keywords(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Keywords) ([]string, error) {

	return func(model.Authorization, txn.GetFilter_Keywords) ([]string, error) {

	}
}

func PostFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_Keyword) (struct{}, error) {

	return func(model.Authorization, txn.PostFilter_Keyword) (struct{}, error) {

	}
}

func GetFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Keyword) (struct{}, error) {

	return func(model.Authorization, txn.GetFilter_Keyword) (struct{}, error) {

	}
}

func PutFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter_Keyword) (struct{}, error) {

	return func(model.Authorization, txn.PutFilter_Keyword) (struct{}, error) {

	}
}

func DeleteFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_Keyword) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_Keyword) (struct{}, error) {

	}
}

func GetFilter_Statuses(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Statuses) ([]object.FilterStatus, error) {

	return func(model.Authorization, txn.GetFilter_Statuses) ([]object.FilterStatus, error) {

	}
}

func PostFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_Status) (object.FilterStatus, error) {

	return func(model.Authorization, txn.PostFilter_Status) (object.FilterStatus, error) {

	}
}

func GetFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Status) (object.FilterStatus, error) {

	return func(model.Authorization, txn.GetFilter_Status) (object.FilterStatus, error) {

	}
}

func DeleteFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_Status) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_Status) (struct{}, error) {

	}
}

func GetFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.GetFilter_V1) (object.Filter, error) {

	}
}

func PostFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.PostFilter_V1) (object.Filter, error) {

	}
}

func PutFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.PutFilter_V1) (object.Filter, error) {

	}
}

func DeleteFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_V1) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_V1) (struct{}, error) {

	}
}
