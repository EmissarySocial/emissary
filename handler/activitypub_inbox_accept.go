package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/hannibal/jsonld"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeAccept, vocab.Any, func(factory *domain.Factory, activity jsonld.Reader) error {
		return nil
	})
}
