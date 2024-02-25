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
func (service *Stream) Publish(user *model.User, stream *model.Stream) error {

	const location = "service.Stream.Publish"

	// RULE: User must be a valid User
	if user.IsNew() {
		return derp.NewForbiddenError(location, "User is not valid", user)
	}

	// RULE: Stream must be a valid Stream
	if stream.IsNew() {
		return derp.NewBadRequestError(location, "Stream is not valid", stream)
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
		return derp.Wrap(err, location, "Error saving stream", stream)
	}

	// Create the Activity to send to the User's Outbox
	object := service.JSONLD(stream)

	// Save the object to the ActivityStream cache
	service.activityService.Put(
		service.activityService.NewDocument(object),
	)

	// Create the Activity to send to Followers
	activityType := iif(stream.IsPublished(), vocab.ActivityTypeUpdate, vocab.ActivityTypeCreate)

	activity := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
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

	return nil
}

// publish_User publishes this stream to the User's outbox
func (service *Stream) publish_User(user *model.User, activity mapof.Any) error {

	const location = "service.Stream.publish_User"

	// Load the Actor for this User
	actor, err := service.userService.ActivityPubActor(user.UserID, true)

	if err != nil {
		return derp.Wrap(err, location, "Error loading actor", user.UserID)
	}

	// Try to publish via sendNotifications
	log.Trace().Str("id", activity.GetString(vocab.PropertyID)).Msg("Publishing to User's outbox")
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

	// Try to publish via sendNotifications
	log.Trace().Str("id", stream.URL).Msg("Publishing to parent Stream's outbox")
	if err := service.outboxService.Publish(&actor, model.FollowerTypeStream, stream.StreamID, activity); err != nil {
		return derp.Wrap(err, location, "Error publishing activity", activity)
	}

	// Done.
	return nil
}

/******************************************
 * UnPublish Methods
 ******************************************/

// UnPublish marks this stream as "published"
func (service *Stream) UnPublish(user *model.User, stream *model.Stream) error {

	// RULE: Move unpublish date all the way to the end of time.
	stream.UnPublishDate = time.Now().Unix()

	// Re-save the Stream with the updated values.
	if err := service.Save(stream, "UnPublish"); err != nil {
		return derp.Wrap(err, "service.Stream.UnPublish", "Error saving stream", stream)
	}

	// Send "Undo" activities to all User followers.
	if err := service.unpublish_User(user.UserID, stream.URL); err != nil {
		return derp.Wrap(err, "service.Stream.UnPublish", "Error unpublishing from User's outbox", stream)
	}

	// Send "Undo" activities to all Stream followers.
	if err := service.unpublish_Stream(stream); err != nil {
		return derp.Wrap(err, "service.Stream.UnPublish", "Error unpublishing from User's outbox", stream)
	}

	// Done.
	return nil
}

// publish_User publishes this stream to the User's outbox
func (service *Stream) unpublish_User(userID primitive.ObjectID, url string) error {

	const location = "service.Stream.publish_User"

	// Load the Actor for this User
	actor, err := service.userService.ActivityPubActor(userID, true)

	if err != nil {
		return derp.Wrap(err, location, "Error loading actor", userID)
	}

	// Try to publish via sendNotifications
	log.Trace().Str("id", url).Msg("UnPublishing from User's outbox")
	if err := service.outboxService.UnPublish(&actor, model.FollowerTypeUser, userID, url); err != nil {
		return derp.Wrap(err, location, "Error un-publishing activity", url)
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
	if err := service.outboxService.UnPublish(&actor, model.FollowerTypeStream, stream.StreamID, stream.ActivityPubURL()); err != nil {
		return derp.Wrap(err, location, "Error publishing activity", stream)
	}

	// Done.
	return nil
}
