package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {

	// This funciton handles ActivityPub "Accept/Follow" activities, meaning that
	// it is called with a remote server accepts our follow request.
	inboxRouter.Add(vocab.Any, vocab.Any, func(factory *domain.Factory, user *model.User, activity streams.Document) error {
		return nil
	})
}
