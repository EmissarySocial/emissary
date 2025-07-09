package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * UnPublish Methods
 ******************************************/

// UnPublish marks this stream as "published"
func (service *Stream) UnPublish(user *model.User, stream *model.Stream, stateID string, outbox bool) error {

	const location = "service.Stream.UnPublish"

	// If we're publishing to the outbox, then do this first, before we modify the Stream.
	if outbox {

		// Send "Undo" activities to all User followers.
		if !user.IsNew() {
			if err := service.unpublish_outbox_user(user.UserID, stream); err != nil {
				return derp.Wrap(err, location, "Unable to unpublish from the User's outbox", stream)
			}
		}

		// Send "Undo" activities to all Stream followers.
		if err := service.unpublish_outbox_stream(stream); err != nil {
			return derp.Wrap(err, location, "Unable to unpublish from parent Stream's outbox", stream)
		}

		// Send stream:publish:undo Webhooks
		service.webhookService.Send(stream, model.WebhookEventStreamPublishUndo)

		// Send syndication:undo messages to all targets
		if err := service.sendSyndicationMessages(stream, nil, nil, stream.Syndication.Values); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to send syndication messages", stream))
		}
	}

	// RULE: Move unpublish date all the way to the end of time.
	stream.StateID = stateID
	stream.UnPublishDate = time.Now().Unix()

	// Re-save the Stream with the updated values.
	if err := service.Save(stream, "UnPublish"); err != nil {
		return derp.Wrap(err, location, "Unable to save the Stream", stream)
	}

	// Done.
	return nil
}

// publish_outbox_user publishes this stream to the User's outbox
func (service *Stream) unpublish_outbox_user(userID primitive.ObjectID, stream *model.Stream) error {

	const location = "service.Stream.unpublish_outbox_user"

	// Load the Actor for this User
	actor, err := service.userService.ActivityPubActor(userID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load actor", userID)
	}

	// Try to publish via sendNotifications
	log.Trace().Str("id", stream.URL).Msg("Publishing a DELETE from User's outbox")
	if err := service.outboxService.DeleteActivity(&actor, model.FollowerTypeUser, userID, stream.URL, stream.DefaultAllow); err != nil {
		return derp.Wrap(err, location, "Unable to unpublish activity", stream.URL)
	}

	// Done.
	return nil
}

// publish_outbox_stream publishes this Stream to the parent Stream's outbox
func (service *Stream) unpublish_outbox_stream(stream *model.Stream) error {

	const location = "service.Stream.unpublish_outbox_stream"

	// RULE: If the Stream does not have a parent template (i.e. Outbox or Top-Level Stream), then NOOP
	if stream.ParentTemplateID == "" {
		return nil
	}

	// Get the parent Template
	parentTemplate, err := service.templateService.Load(stream.ParentTemplateID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load parent template", stream.ParentTemplateID)
	}

	// RULE: If the parent Actor is not set to boost children, then NOOP
	if !parentTemplate.Actor.BoostChildren {
		return nil
	}

	// Load the Actor for the parent Stream
	actor, err := service.ActivityPubActor(stream.ParentID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load parent actor")
	}

	// Try to publish via sendNotifications
	log.Trace().Str("id", stream.URL).Msg("Deleting object from parent's outbox")
	if err := service.outboxService.DeleteActivity(&actor, model.FollowerTypeStream, stream.ParentID, stream.ActivityPubURL(), stream.DefaultAllow); err != nil {
		return derp.Wrap(err, location, "Unable to publish a DELETE activity for this Stream", stream)
	}

	// Done.
	return nil
}
