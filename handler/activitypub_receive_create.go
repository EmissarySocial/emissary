package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/davecgh/go-spew/spew"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.Any, func(factory *domain.Factory, activity streams.Document) error {

		spew.Dump(activity.Value())

		// Hooo-dat?
		return nil
	})
}
