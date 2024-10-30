package service

import (
	"math"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Publish Methods
 ******************************************/

// Publish marks this stream as "published"
func (service *Stream) Publish(user *model.User, stream *model.Stream, outbox bool) error {

	const location = "service.Stream.Publish"

	// Determine ActitivyType FIRST, before we mess with the publish date
	activityType := stream.PublishActivity()

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
		return derp.Wrap(err, location, "Error saving stream", stream)
	}

	// If we're NOT publishing to the outbox, then we're done.
	if !outbox {
		return nil
	}

	// PUBLISH TO THE OUTBOX...

	// Create the Activity to send to the User's Outbox
	object := service.JSONLD(stream)

	// Save the object to the ActivityStream cache
	service.activityStream.Put(
		service.activityStream.NewDocument(object),
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

	// Publish to the User's outbox
	if err := service.publish_User(user, activity); err != nil {
		return derp.Wrap(err, location, "Error publishing to User's outbox")
	}

	// Publish to the parent Stream's outbox
	if err := service.publish_Stream(stream, activity); err != nil {
		return derp.Wrap(err, location, "Error publishing to parent Stream's outbox")
	}

	// Send stream:publish Webhooks
	service.webhookService.Send(stream, model.WebhookEventStreamPublish)

	// Send syndication messages to all targets
	if err := service.sendSyndicationMessages(stream, stream.Syndication.Values, nil); err != nil {
		derp.Report(derp.Wrap(err, location, "Error sending syndication messages", stream))
	}

	return nil
}

// publish_User publishes this stream to the User's outbox
func (service *Stream) publish_User(user *model.User, activity mapof.Any) error {

	const location = "service.Stream.publish_User"

	// Do not take actions on an empty user
	if user.IsNew() {
		return nil
	}

	// Load the Actor for this User
	actor, err := service.userService.ActivityPubActor(user.UserID, true)

	if err != nil {
		return derp.Wrap(err, location, "Error loading actor", user.UserID)
	}

	// Try to publish via sendNotifications
	objectID := activity.GetString(vocab.PropertyID)
	objectType := activity.GetString(vocab.PropertyType)
	log.Trace().Str("location", location).Str("objectId", objectID).Str("type", objectType).Msg("Publishing to User's outbox")

	if err := service.outboxService.Publish(&actor, model.FollowerTypeUser, user.UserID, activity); err != nil {
		return derp.Wrap(err, location, "Error publishing activity", activity)
	}

	// Done.
	return nil
}

// publish_Stream publishes this Stream to the parent Stream's outbox
func (service *Stream) publish_Stream(stream *model.Stream, activity mapof.Any) error {

	const location = "service.Stream.publish_Stream"

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
	actor, err := service.ActivityPubActor(stream.ParentID, true)

	if err != nil {
		return derp.Wrap(err, location, "Error loading parent actor")
	}

	// Make a new "Announce/Boost" activity so that our encryption keys are correct.
	boostActivity := mapof.Any{
		vocab.AtContext:      vocab.ContextTypeActivityStreams,
		vocab.PropertyType:   vocab.ActivityTypeAnnounce,
		vocab.PropertyActor:  service.ActivityPubURL(stream.ParentID),
		vocab.PropertyObject: activity,
	}

	// Try to publish via sendNotifications
	log.Trace().Str("id", stream.URL).Msg("Publishing to parent Stream's outbox")
	if err := service.outboxService.Publish(&actor, model.FollowerTypeStream, stream.ParentID, boostActivity); err != nil {
		return derp.Wrap(err, location, "Error publishing activity", activity)
	}

	// Done.
	return nil
}

/******************************************
 * UnPublish Methods
 ******************************************/

// UnPublish marks this stream as "published"
func (service *Stream) UnPublish(user *model.User, stream *model.Stream, outbox bool) error {

	const location = "service.Stream.UnPublish"

	// RULE: Move unpublish date all the way to the end of time.
	stream.UnPublishDate = time.Now().Unix()

	// Re-save the Stream with the updated values.
	if err := service.Save(stream, "UnPublish"); err != nil {
		return derp.Wrap(err, location, "Error saving stream", stream)
	}

	// If we're not publishing to the outbox, then we're done
	if !outbox {
		return nil
	}

	// UN-PUBLISH FROM THE OUTBOX...

	// Send "Undo" activities to all User followers.
	if !user.IsNew() {
		if err := service.unpublish_User(user.UserID, stream.URL); err != nil {
			return derp.Wrap(err, location, "Error unpublishing from User's outbox", stream)
		}
	}

	// Send "Undo" activities to all Stream followers.
	if err := service.unpublish_Stream(stream); err != nil {
		return derp.Wrap(err, location, "Error unpublishing from User's outbox", stream)
	}

	// Send stream:publish:undo Webhooks
	service.webhookService.Send(stream, model.WebhookEventStreamPublishUndo)

	// Send syndication:undo messages to all targets
	if err := service.sendSyndicationMessages(stream, nil, stream.Syndication.Values); err != nil {
		derp.Report(derp.Wrap(err, location, "Error sending syndication messages", stream))
	}

	// Done.
	return nil
}

// publish_User publishes this stream to the User's outbox
func (service *Stream) unpublish_User(userID primitive.ObjectID, url string) error {

	const location = "service.Stream.unpublish_User"

	// Load the Actor for this User
	actor, err := service.userService.ActivityPubActor(userID, true)

	if err != nil {
		return derp.Wrap(err, location, "Error loading actor", userID)
	}

	// Try to publish via sendNotifications
	log.Trace().Str("id", url).Msg("UnPublishing from User's outbox")
	if err := service.outboxService.UnPublish(&actor, model.FollowerTypeUser, userID, url); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Error un-publishing activity", url))
	}

	// Done.
	return nil
}

// publish_Stream publishes this Stream to the parent Stream's outbox
func (service *Stream) unpublish_Stream(stream *model.Stream) error {

	const location = "service.Stream.unpublish_Stream"

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
	actor, err := service.ActivityPubActor(stream.ParentID, true)

	if err != nil {
		return derp.Wrap(err, location, "Error loading parent actor")
	}

	// Try to publish via sendNotifications
	log.Trace().Str("id", stream.URL).Msg("UnPublishing from parent Stream's outbox")
	if err := service.outboxService.UnPublish(&actor, model.FollowerTypeStream, stream.ParentID, stream.ActivityPubURL()); err != nil {
		return derp.Wrap(err, location, "Error publishing activity", stream)
	}

	// Done.
	return nil
}
