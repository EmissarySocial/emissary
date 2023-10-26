package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/lists/
func GetLists(serverFactory *server.Factory) func(model.Authorization, txn.GetLists) ([]object.List, error) {

	return func(model.Authorization, txn.GetLists) ([]object.List, error) {

	}
}

func GetList(serverFactory *server.Factory) func(model.Authorization, txn.GetList) (object.List, error) {

	return func(model.Authorization, txn.GetList) (object.List, error) {

	}
}

func PostList(serverFactory *server.Factory) func(model.Authorization, txn.PostList) (object.List, error) {

	return func(model.Authorization, txn.PostList) (object.List, error) {

	}
}

func PutList(serverFactory *server.Factory) func(model.Authorization, txn.PutList) (object.List, error) {

	return func(model.Authorization, txn.PutList) (object.List, error) {

	}
}

func DeleteList(serverFactory *server.Factory) func(model.Authorization, txn.DeleteList) (struct{}, error) {

	return func(model.Authorization, txn.DeleteList) (struct{}, error) {

	}
}

func GetList_Accounts(serverFactory *server.Factory) func(model.Authorization, txn.GetList_Accounts) ([]object.Account, error) {

	return func(model.Authorization, txn.GetList_Accounts) ([]object.Account, error) {

	}
}

func PostList_Accounts(serverFactory *server.Factory) func(model.Authorization, txn.PostList_Accounts) (struct{}, error) {

	return func(model.Authorization, txn.PostList_Accounts) (struct{}, error) {

	}
}
func DeleteList_Accounts(serverFactory *server.Factory) func(model.Authorization, txn.DeleteList_Accounts) (struct{}, error) {

	return func(model.Authorization, txn.DeleteList_Accounts) (struct{}, error) {

	}
}
