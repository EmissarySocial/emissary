package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/conversations/

func GetConversations(serverFactory *server.Factory) func(model.Authorization, txn.GetConversations) ([]object.Conversation, error) {

	return func(auth model.Authorization, t txn.GetConversations) ([]object.Conversation, error) {
		return []object.Conversation{}, nil
	}
}
func DeleteConversation(serverFactory *server.Factory) func(model.Authorization, txn.DeleteConversation) (struct{}, error) {

	return func(auth model.Authorization, t txn.DeleteConversation) (struct{}, error) {
		return struct{}{}, nil
	}
}

func PostConversationRead(serverFactory *server.Factory) func(model.Authorization, txn.PostConversationRead) (struct{}, error) {

	return func(auth model.Authorization, t txn.PostConversationRead) (struct{}, error) {
		return struct{}{}, nil
	}
}
