package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeDislike, vocab.Any, func(factory *domain.Factory, user *model.User, activity streams.Document) error {
		// If we receive a DISLIKE, NO-OP for now.

		// Eventually, we may update some statistics on this object, or down-rank it in the feed.
		return nil
	})
}
