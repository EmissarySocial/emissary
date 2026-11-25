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
		if err := service.sendSyndicationMessages(session, stream, nil, nil, stream.Syndication.Value()); err != nil {
			derp.Report(derp.Wrap(err, location, "Sending syndication messages", stream))
		}
	}

	// Set the new state, and end publication now
	stream.StateID.Set(stateID)
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

	// RULE: Send a Delete only while the author's outbox still holds a Create or Update of the Stream.
	// Unpublishing removes those and leaves only its own Delete, and deleting the author removes
	// every row, so neither one sends a second Delete.
	federated, err := service.isFederated(session, authorID, stream.URL)

	if err != nil {
		return derp.Wrap(err, location, "Checking the author's outbox", stream.URL)
	}

	if !federated {
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

// isFederated returns TRUE if the author's legacy outbox or Outbox2 still holds an activity that publishes the object
func (service *Stream) isFederated(session data.Session, authorID primitive.ObjectID, objectID string) (bool, error) {

	const location = "service.Stream.isFederated"

	// Ask the legacy outbox, which holds every Stream published so far
	published, err := service.outboxService.HasObject(session, model.ActorTypeUser, authorID, objectID)

	if err != nil {
		return false, derp.Wrap(err, location, "Checking the legacy outbox", objectID)
	}

	if published {
		return true, nil
	}

	// Ask Outbox2 too, because both outboxes are read until the migration moves every row
	published, err = service.outbox2Service.HasObject(session, model.ActorTypeUser, authorID, objectID)

	if err != nil {
		return false, derp.Wrap(err, location, "Checking Outbox2", objectID)
	}

	return published, nil
}

// unpublish_outbox_user sends a Delete for this Stream from the User's outbox
func (service *Stream) unpublish_outbox_user(session data.Session, userID primitive.ObjectID, stream *model.Stream) error {

	const location = "service.Stream.unpublish_outbox_user"

	// Send the Delete from the User's outbox. It copies the addressing of the Stream's Create, and a
	// Stream that was published before Outbox2 has none, so it falls back to the Stream's audience.
	log.Trace().Str("id", stream.URL).Msg("Publishing a DELETE from User's outbox")
	if err := service.outbox2Service.DeleteActivity(session, model.ActorTypeUser, userID, stream.URL, stream.DefaultAllow, service.audienceMap(stream)); err != nil {
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

	// Send the Delete from the parent Stream's outbox, to the audience that the parent announced it to
	if err := service.outbox2Service.DeleteActivity(session, model.ActorTypeStream, stream.ParentID, stream.ActivityPubURL(), stream.DefaultAllow, service.parentAudience(stream)); err != nil {
		return derp.Wrap(err, location, "Publishing a DELETE activity for this Stream", stream.URL)
	}

	// Done.
	return nil
}
