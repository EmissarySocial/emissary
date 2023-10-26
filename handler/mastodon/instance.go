package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/instance/
func GetInstance(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance) (object.Instance, error) {

	return func(model.Authorization, txn.GetInstance) (object.Instance, error) {

	}
}

func GetInstance_Peers(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance_Peers) ([]string, error) {

	return func(model.Authorization, txn.GetInstance_Peers) ([]string, error) {

	}
}

func GetInstance_Activity(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance_Activity) (map[string]any, error) {

	return func(model.Authorization, txn.GetInstance_Activity) (map[string]any, error) {

	}
}

func GetInstance_Rules(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance_Rules) ([]object.Rule, error) {

	return func(model.Authorization, txn.GetInstance_Rules) ([]object.Rule, error) {

	}
}

func GetInstance_DomainBlocks(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance_DomainBlocks) ([]object.DomainBlock, error) {

	return func(model.Authorization, txn.GetInstance_DomainBlocks) ([]object.DomainBlock, error) {

	}
}

func GetInstance_ExtendedDescription(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance_ExtendedDescription) (object.ExtendedDescription, error) {

	return func(model.Authorization, txn.GetInstance_ExtendedDescription) (object.ExtendedDescription, error) {

	}
}
