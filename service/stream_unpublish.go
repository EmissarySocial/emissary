package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * UnPublish Methods
 ******************************************/

// UnPublish moves this Stream into the provided state.  The save queues the outbox task,
// which is what tells followers and syndication targets.
func (service *Stream) UnPublish(session data.Session, stream *model.Stream, stateID string) error {

	const location = "service.Stream.UnPublish"

	// Set the new state.  The save takes the Stream out of the outbox when the state says so.
	stream.StateID.Set(stateID)

	// Re-save the Stream with the updated values.
	if err := service.Save(session, stream, "UnPublish"); err != nil {
		return derp.Wrap(err, location, "Saving the Stream", stream.StreamID)
	}

	// Done.
	return nil
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
