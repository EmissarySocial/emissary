package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/polls/
func GetPoll(serverFactory *server.Factory) func(model.Authorization, txn.GetPoll) ([]object.Poll, error) {

	return func(model.Authorization, txn.GetPoll) ([]object.Poll, error) {
		return nil, errNoSuch("handler.mastodon.GetPoll", "Poll")
	}
}

// PostPoll_Votes is the Mastodon "vote on a poll" endpoint, which Emissary does not implement
func PostPoll_Votes(serverFactory *server.Factory) func(model.Authorization, txn.PostPoll_Votes) ([]object.Poll, error) {

	return func(model.Authorization, txn.PostPoll_Votes) ([]object.Poll, error) {
		return nil, errNoSuch("handler.mastodon.PostPoll_Votes", "Poll")
	}
}
