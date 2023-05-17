package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeLike, undoResponse)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeLike, undoResponse)

	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeDislike, undoResponse)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeDislike, undoResponse)

	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeAnnounce, undoResponse)
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActivityTypeAnnounce, undoResponse)
}

// undoResponse handles the Undo/Delete actions on Like/Dislike/Announce records
func undoResponse(factory *domain.Factory, user *model.User, document streams.Document) error {
	// Hooo-dat?!?!?
	return nil
}
