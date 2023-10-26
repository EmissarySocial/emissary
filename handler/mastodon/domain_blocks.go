package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/domain_blocks/
func GetDomainBlocks(serverFactory *server.Factory) func(model.Authorization, txn.GetDomainBlocks) ([]string, error) {

	return func(model.Authorization, txn.GetDomainBlocks) ([]string, error) {

	}
}

func PostDomainBlock(serverFactory *server.Factory) func(model.Authorization, txn.PostDomainBlock) (struct{}, error) {

	return func(model.Authorization, txn.PostDomainBlock) (struct{}, error) {

	}
}

func DeleteDomainBlock(serverFactory *server.Factory) func(model.Authorization, txn.DeleteDomainBlock) (struct{}, error) {

	return func(model.Authorization, txn.DeleteDomainBlock) (struct{}, error) {

	}
}
