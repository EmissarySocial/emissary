package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/follow_requests/
func GetFollowRequests(serverFactory *server.Factory) func(model.Authorization, txn.GetFollowRequests) ([]object.Account, error) {

	return func(model.Authorization, txn.GetFollowRequests) ([]object.Account, error) {
		return []object.Account{}, nil
	}
}

func PostFollowRequest_Authorize(serverFactory *server.Factory) func(model.Authorization, txn.PostFollowRequest_Authorize) (object.Relationship, error) {

	return func(model.Authorization, txn.PostFollowRequest_Authorize) (object.Relationship, error) {
		return object.Relationship{}, derp.NewBadRequestError("handler.mastodon.PostFollowRequest_Authorize", "Not implemented")
	}
}

func PostFollowRequest_Reject(serverFactory *server.Factory) func(model.Authorization, txn.PostFollowRequest_Reject) (object.Relationship, error) {

	return func(model.Authorization, txn.PostFollowRequest_Reject) (object.Relationship, error) {
		return object.Relationship{}, derp.NewBadRequestError("handler.mastodon.PostFollowRequest_Reject", "Not implemented")
	}
}
