package service

import (
	"math"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/rs/zerolog/log"
)

// Publish marks this stream as "published"
func (service *Stream) Publish(user *model.User, stream *model.Stream) error {

	activityType := vocab.ActivityTypeCreate

	if stream.IsPublished() {
		activityType = vocab.ActivityTypeUpdate
	}

	// RULE: IF this stream is not yet published, then set the publish date
	if stream.PublishDate > time.Now().Unix() {
		stream.PublishDate = time.Now().Unix()
	}

	// RULE: Move unpublish date all the way to the end of time.
	// TODO: LOW: May want to set automatic unpublish dates later...
	stream.UnPublishDate = math.MaxInt64

	// RULE: Set Author to the currently logged in user.
	stream.SetAttributedTo(user.PersonLink())

	// Re-save the Stream with the updated values.
	if err := service.Save(stream, "Publishing"); err != nil {
		return derp.Wrap(err, "service.Stream.Publish", "Error saving stream", stream)
	}

	// Attempt to pre-load the ActivityStream cache.  We don't care about the result.
	_, _ = service.activityService.Load(stream.ActivityPubURL())

	object := service.JSONLD(stream)

	// Create the Activity to send to the User's Outbox
	activity := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyID:        stream.ActivityPubURL(),
		vocab.PropertyType:      activityType,
		vocab.PropertyActor:     user.ActivityPubURL(),
		vocab.PropertyObject:    object,
		vocab.PropertyPublished: time.Now().UTC().Format(time.RFC3339),
	}

	if to, ok := object[vocab.PropertyTo]; ok {
		activity[vocab.PropertyTo] = to
	}

	if cc, ok := object[vocab.PropertyCC]; ok {
		activity[vocab.PropertyCC] = cc
	}

	// Try to publish via the outbox service
	log.Trace().Msg("Publishing stream: " + stream.URL)
	if err := service.outboxService.Publish(user.UserID, stream.URL, activity); err != nil {
		return derp.Wrap(err, "service.Stream.Publish", "Error publishing activity", activity)
	}

	// Done.
	return nil
}

// UnPublish marks this stream as "published"
func (service *Stream) UnPublish(user *model.User, stream *model.Stream) error {

	// RULE: Move unpublish date all the way to the end of time.
	stream.UnPublishDate = time.Now().Unix()

	// Re-save the Stream with the updated values.
	if err := service.Save(stream, "Publish"); err != nil {
		return derp.Wrap(err, "service.Stream.UnPublish", "Error saving stream", stream)
	}

	// Create the Activity to send to the User's Outbox
	activity := mapof.Any{
		vocab.AtContext:      vocab.ContextTypeActivityStreams,
		vocab.PropertyType:   vocab.ActivityTypeDelete,
		vocab.PropertyActor:  user.ActivityPubURL(),
		vocab.PropertyObject: service.JSONLD(stream),
	}

	// Remove the record from the inbox
	if err := service.outboxService.UnPublish(user.UserID, stream.URL, activity); err != nil {
		return derp.Wrap(err, "service.Stream.UnPublish", "Error removing from outbox", stream)
	}

	// Done.
	return nil
}
