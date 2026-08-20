package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/conversations/

// GetConversations implements the Mastodon "get conversations" endpoint, and always returns an empty list
func GetConversations(serverFactory *server.Factory) func(model.Authorization, txn.GetConversations) ([]object.Conversation, toot.PageInfo, error) {

	return func(auth model.Authorization, t txn.GetConversations) ([]object.Conversation, toot.PageInfo, error) {
		return []object.Conversation{}, toot.PageInfo{}, nil
	}
}

// DeleteConversation implements the Mastodon "delete conversation" endpoint as a no-op
func DeleteConversation(serverFactory *server.Factory) func(model.Authorization, txn.DeleteConversation) (struct{}, error) {

	return func(auth model.Authorization, t txn.DeleteConversation) (struct{}, error) {
		return struct{}{}, nil
	}
}

// PostConversationRead implements the Mastodon "mark conversation read" endpoint as a no-op
func PostConversationRead(serverFactory *server.Factory) func(model.Authorization, txn.PostConversationRead) (struct{}, error) {

	return func(auth model.Authorization, t txn.PostConversationRead) (struct{}, error) {
		return struct{}{}, nil
	}
}
