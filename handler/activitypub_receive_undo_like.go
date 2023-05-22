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

	// Try to load the the object of the original activity (the local Stream that was liked/dislked/announced)
	streamService := factory.Stream()
	stream := model.NewStream()
	streamURL := originalActivity.Object().ID()

	if err := streamService.LoadByURL(streamURL, &stream); err != nil {
		return derp.Wrap(err, "handler.undoResponse", "Error loading Stream")
	}

	// Try to load the StreamResponse that matches the Stream and origin
	streamResponseService := factory.StreamResponse()
	streamResponse := model.NewStreamResponse()
	if err := streamResponseService.LoadByStreamAndOrigin(stream.StreamID, originalActivity.ID(), &streamResponse); err != nil {
		return derp.Wrap(err, "handler.undoResponse", "Error loading StreamResponse")
	}

	// RULE: ActivityPub type must match the received activity
	if streamResponse.ActivityPubType() != originalActivity.Type() {
		return derp.New(derp.CodeBadRequestError, "handler.undoResponse", "ActivityPub type does not match")
	}

	// RULE: The Actor must match the received activity
	if streamResponse.Actor.ProfileURL != activity.Actor().String() {
		return derp.New(derp.CodeBadRequestError, "handler.undoResponse", "Actor does not match")
	}

	// Delete the StreamResponse
	if err := streamResponseService.Delete(&streamResponse, "Undo via ActivityPub"); err != nil {
		return derp.Wrap(err, "handler.receiveResponse", "Error deleting StreamResponse")
	}

	return nil
}
