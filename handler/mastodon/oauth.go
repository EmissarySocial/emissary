package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/oauth/
func GetOAuth_Authorize(serverFactory *server.Factory) func(model.Authorization, txn.GetOAuth_Authorize) (struct{}, error) {

	return func(model.Authorization, txn.GetOAuth_Authorize) (struct{}, error) {

	}
}

func PostOAuth_Token(serverFactory *server.Factory) func(model.Authorization, txn.PostOAuth_Token) (object.Token, error) {

	return func(model.Authorization, txn.PostOAuth_Token) (object.Token, error) {

	}
}

func PostOAuth_Revoke(serverFactory *server.Factory) func(model.Authorization, txn.PostOAuth_Revoke) (struct{}, error) {

	return func(model.Authorization, txn.PostOAuth_Revoke) (struct{}, error) {

	}
}
