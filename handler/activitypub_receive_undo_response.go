package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
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
func undoResponse(factory *domain.Factory, user *model.User, activity streams.Document) error {

	// Try to parse the original Activity from the JSON-LD
	originalActivity, err := activity.Object().Load() // The Object is the original Like/Dislike/Announce activity

	if err != nil {
		return derp.Wrap(err, "handler.undoResponse", "Error loading originalActivity")
	}

	responseService := factory.Response()
	response := model.NewResponse()

	// Try to load the Response that matches the original activity
	if err := responseService.LoadByActorAndObject(originalActivity.Actor().ID(), originalActivity.Object().ID(), &response); err != nil {
		return derp.Wrap(err, "handler.undoResponse", "Error loading Response")
	}

	// RULE: ActivityPub type must match the received activity
	if activity.Actor().ID() != response.ActorID {
		return derp.NewUnauthorizedError("handler.undoResponse", "Actor does not match")
	}

	// RULE: ActivityPub type must match the received activity
	if response.ActivityPubType() != originalActivity.Type() {
		return derp.New(derp.CodeBadRequestError, "handler.undoResponse", "ActivityPub type does not match")
	}

	// Try to remove the Response from the database. (This will NOT send updates to other servers)
	if err := responseService.Delete(&response, "Undo via ActivityPub"); err != nil {
		return derp.Wrap(err, "handler.undoResponse", "Error deleting Response")
	}

	return nil
}
