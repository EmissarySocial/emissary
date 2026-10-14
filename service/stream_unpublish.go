package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * UnPublish Methods
 ******************************************/

// UnPublish ends this Stream's publication, and tells its followers and syndication targets when `outbox` is TRUE
func (service *Stream) UnPublish(session data.Session, user *model.User, stream *model.Stream, stateID string, outbox bool) error {

	const location = "service.Stream.UnPublish"

	// Unpublish from the outboxes first, before the Stream is modified
	if outbox {

		// Send a Delete to the User's followers
		if !user.IsNew() {
			if err := service.unpublish_outbox_user(session, user.UserID, stream); err != nil {
				return derp.Wrap(err, location, "Unpublishing from the User's outbox", stream)
			}
		}

		// Send a Delete to the parent Stream's followers
		if err := service.unpublish_outbox_stream(session, stream); err != nil {
			return derp.Wrap(err, location, "Unpublishing from parent Stream's outbox", stream)
		}

		// Send stream:publish:undo Webhooks
		service.webhookService.Send(stream, model.WebhookEventStreamPublishUndo)

		// Queue a syndication.delete message for each target
		if err := service.sendSyndicationMessages(session, stream, nil, nil, stream.Syndication.Values); err != nil {
			derp.Report(derp.Wrap(err, location, "Sending syndication messages", stream))
		}
	}

	// Set the new state, and end publication now
	stream.StateID = stateID
	stream.UnPublishDate = time.Now().Unix()

	// Re-save the Stream with the updated values.
	if err := service.Save(session, stream, "UnPublish"); err != nil {
		return derp.Wrap(err, location, "Saving the Stream", stream)
	}

	// Done.
	return nil
}

// federateDelete sends a Delete for a deleted Stream wherever its author's outbox published it
func (service *Stream) federateDelete(session data.Session, stream *model.Stream) error {

	const location = "service.Stream.federateDelete"

	// RULE: A Stream with no author was never published to a User's outbox
	authorID := stream.AttributedTo.UserID

	if authorID.IsZero() {
		return nil
	}

	// RULE: Send a Delete only while the author's outbox still holds the Stream.  Unpublishing
	// removes it, and so does deleting the author, so neither one sends a second Delete.
	published, err := service.outboxService.HasObject(session, model.FollowerTypeUser, authorID, stream.URL)

	if err != nil {
		return derp.Wrap(err, location, "Checking the author's outbox", stream.URL)
	}

	if !published {
		return nil
	}

	// Send the Delete to the author's followers
	if err := service.unpublish_outbox_user(session, authorID, stream); err != nil {
		return derp.Wrap(err, location, "Unpublishing from the User's outbox", stream.URL)
	}

	// Send the Delete to the parent Stream's followers
	if err := service.unpublish_outbox_stream(session, stream); err != nil {
		return derp.Wrap(err, location, "Unpublishing from the parent Stream's outbox", stream.URL)
	}

	// Gone with the wind
	return nil
}

// unpublish_outbox_user sends a Delete for this Stream from the User's outbox
func (service *Stream) unpublish_outbox_user(session data.Session, userID primitive.ObjectID, stream *model.Stream) error {

	const location = "service.Stream.unpublish_outbox_user"

	// Send the Delete from the User's outbox
	log.Trace().Str("id", stream.URL).Msg("Publishing a DELETE from User's outbox")
	if err := service.outboxService.DeleteActivity(session, model.FollowerTypeUser, userID, stream.URL, stream.DefaultAllow); err != nil {
		return derp.Wrap(err, location, "Unpublishing activity", stream.URL)
	}

	// Done.
	return nil
}

// unpublish_outbox_stream sends a Delete for this Stream from the parent Stream's outbox, when the parent boosts its children
func (service *Stream) unpublish_outbox_stream(session data.Session, stream *model.Stream) error {

	const location = "service.Stream.unpublish_outbox_stream"

	// RULE: If the Stream does not have a parent template (i.e. Outbox or Top-Level Stream), then NOOP
	if stream.ParentTemplateID == "" {
		return nil
	}

	// Get the parent Template
	parentTemplate, err := service.templateService.Load(stream.ParentTemplateID)

	if err != nil {
		return derp.Wrap(err, location, "Loading parent template", stream.ParentTemplateID)
	}

	// RULE: If the parent Actor is not set to boost children, then NOOP
	if !parentTemplate.Actor.BoostChildren {
		return nil
	}

	// Send the Delete from the parent Stream's outbox
	if err := service.outboxService.DeleteActivity(session, model.FollowerTypeStream, stream.ParentID, stream.ActivityPubURL(), stream.DefaultAllow); err != nil {
		return derp.Wrap(err, location, "Publishing a DELETE activity for this Stream", stream)
	}

	// Done.
	return nil
}
