package service

import (
	"math"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
	"github.com/rs/zerolog/log"
)

/******************************************
 * Publish Methods
 ******************************************/

// Publish marks this stream as "published"
func (service *Stream) Publish(user *model.User, stream *model.Stream, stateID string, outbox bool, republish bool) error {

	const location = "service.Stream.Publish"

	wasPublished := stream.IsPublished()

	// RULE: IF this stream is not yet published, then set the publish date
	if (stream.PublishDate > time.Now().Unix()) || (stream.StateID != stateID) {
		stream.PublishDate = time.Now().Unix()
	}

	// RULE: Move unpublish date all the way to the end of time.
	// TODO: LOW: May want to set automatic unpublish dates later...
	stream.UnPublishDate = math.MaxInt64

	// RULE: Set Author to the currently logged in user.
	stream.SetAttributedTo(user.PersonLink())

	// RULE: Set the new state ID
	stream.StateID = stateID

	// Re-save the Stream with the updated values.
	if err := service.Save(stream, "Publishing"); err != nil {
		return derp.Wrap(err, location, "Error saving stream", stream)
	}

	// Publish to user/stream outboxes
	if outbox {
		if err := service.publish_outbox(user, stream, wasPublished); err != nil {
			return derp.Wrap(err, location, "Error publishing to outbox", stream)
		}
	}

	// Send stream:publish Webhooks
	service.webhookService.Send(stream, model.WebhookEventStreamPublish)

	// Send syndication messages to all targets
	switch {

	// If the stream is being published for the first time, then only send "Create" activities
	case !wasPublished:
		if err := service.sendSyndicationMessages(stream, stream.Syndication.Values, nil, nil); err != nil {
			return derp.Wrap(err, location, "Error sending syndication messages", stream)
		}

	// If the syndication settings have been changed (or is being republished) then send "Update" activities
	case stream.Syndication.IsChanged() || republish:

		if err := service.sendSyndicationMessages(stream, stream.Syndication.Added, stream.Syndication.Unchanged(), stream.Syndication.Deleted); err != nil {
			return derp.Wrap(err, location, "Error sending syndication messages", stream)
		}
	}

	return nil
}

func (service *Stream) publish_outbox(user *model.User, stream *model.Stream, wasPublished bool) error {

	const location = "service.Stream.publish_outbox"

	// Create the Activity to send to the User's Outbox
	object := service.JSONLD(stream)

	// Save the object to the ActivityStream cache
	service.activityStream.Put(
		service.activityStream.NewDocument(object),
	)

	// If this has not been published yet, then `Create` activity. Otherwise, `Update`
	activityType := iif(
		wasPublished,
		vocab.ActivityTypeUpdate,
		vocab.ActivityTypeCreate,
	)

	// Create the Activity to send to Followers
	activity := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyID:        stream.ActivityPubURL(),
		vocab.PropertyType:      activityType,
		vocab.PropertyActor:     user.ActivityPubURL(),
		vocab.PropertyObject:    object,
		vocab.PropertyPublished: hannibal.TimeFormat(time.Now()),
	}

	if to, ok := object[vocab.PropertyTo]; ok {
		activity[vocab.PropertyTo] = to
	}

	if cc, ok := object[vocab.PropertyCC]; ok {
		activity[vocab.PropertyCC] = cc
	}

	spew.Dump(location, activity)

	// Publish to the User's outbox
	if err := service.publish_outbox_user(user, stream, activity); err != nil {
		return derp.Wrap(err, location, "Error publishing to User's outbox")
	}

	// Publish to the parent Stream's outbox
	if err := service.publish_outbox_stream(stream, activity); err != nil {
		return derp.Wrap(err, location, "Error publishing to parent Stream's outbox")
	}

	return nil
}

// publish_outbox_user publishes this stream to the User's outbox
func (service *Stream) publish_outbox_user(user *model.User, stream *model.Stream, activity mapof.Any) error {

	const location = "service.Stream.publish_outbox_user"

	// RULE: Do not allow empty Users
	if user == nil {
		return derp.InternalError(location, "User cannot be nil")
	}

	// RULE: Do not allow "new" Users
	if user.IsNew() {
		return nil
	}

	// Load the Actor for this User
	actor, err := service.userService.ActivityPubActor(user.UserID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading actor", user.UserID)
	}

	// Try to publish via sendNotifications
	objectID := activity.GetString(vocab.PropertyID)
	objectType := activity.GetString(vocab.PropertyType)
	log.Trace().Str("location", location).Str("objectId", objectID).Str("type", objectType).Msg("Publishing to User's outbox")

	document := service.activityStream.NewDocument(activity)

	if err := service.outboxService.Publish(&actor, model.FollowerTypeUser, user.UserID, document, stream.DefaultAllow); err != nil {
		return derp.Wrap(err, location, "Error publishing activity", activity)
	}

	// Done.
	return nil
}

// publish_outbox_stream publishes this Stream to the parent Stream's outbox
func (service *Stream) publish_outbox_stream(stream *model.Stream, activity mapof.Any) error {

	const location = "service.Stream.publish_outbox_stream"

	// RULE: If the Stream does not have a parent template (i.e. Outbox or Top-Level Stream), then NOOP
	if stream.ParentTemplateID == "" {
		return nil
	}

	// Get the parent Template
	parentTemplate, err := service.templateService.Load(stream.ParentTemplateID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading parent template", stream.ParentTemplateID)
	}

	// RULE: If the parent Actor is not set to boost children, then NOOP
	if !parentTemplate.Actor.BoostChildren {
		return nil
	}

	// Load the Actor for the parent Stream
	actor, err := service.ActivityPubActor(stream.ParentID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading parent actor")
	}

	// Make a new "Announce/Boost" activity so that our encryption keys are correct.
	announce := mapof.Any{
		vocab.AtContext:      vocab.ContextTypeActivityStreams,
		vocab.PropertyType:   vocab.ActivityTypeAnnounce,
		vocab.PropertyActor:  service.ActivityPubURL(stream.ParentID),
		vocab.PropertyObject: activity,
	}

	document := service.activityStream.NewDocument(announce)

	// Try to publish via sendNotifications
	log.Trace().Str("id", stream.URL).Msg("Publishing to parent Stream's outbox")
	if err := service.outboxService.Publish(&actor, model.FollowerTypeStream, stream.ParentID, document, stream.DefaultAllow); err != nil {
		return derp.Wrap(err, location, "Error publishing activity", activity)
	}

	// Done.
	return nil
}
