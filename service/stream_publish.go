package service

import (
	"math"
	"slices"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/datetime"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/rs/zerolog/log"
)

/******************************************
 * Publish Methods
 ******************************************/

// Publish marks this stream as "published"
func (service *Stream) Publish(session data.Session, user *model.User, stream *model.Stream, stateID string, outbox bool, republish bool) error {

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
	if err := service.Save(session, stream, "Publishing"); err != nil {
		return derp.Wrap(err, location, "Saving stream", stream)
	}

	// Publish to user/stream outboxes
	if outbox {
		if err := service.publish_outbox(session, user, stream, wasPublished); err != nil {
			return derp.Wrap(err, location, "Publishing to outbox", stream)
		}
	}

	// Send stream:publish Webhooks
	service.webhookService.Send(stream, model.WebhookEventStreamPublish)

	// Send syndication messages to all targets
	switch {

	// If the stream is being published for the first time, then only send "Create" activities
	case !wasPublished:
		if err := service.sendSyndicationMessages(session, stream, stream.Syndication.Values, nil, nil); err != nil {
			return derp.Wrap(err, location, "Sending syndication messages", stream)
		}

	// If the syndication settings have been changed (or is being republished) then send "Update" activities
	case stream.Syndication.IsChanged() || republish:

		if err := service.sendSyndicationMessages(session, stream, stream.Syndication.Added, stream.Syndication.Unchanged(), stream.Syndication.Deleted); err != nil {
			return derp.Wrap(err, location, "Sending syndication messages", stream)
		}
	}

	return nil
}

func (service *Stream) publish_outbox(session data.Session, user *model.User, stream *model.Stream, wasPublished bool) error {

	const location = "service.Stream.publish_outbox"

	// Create the Activity to send to the User's Outbox.  @mentions were already extracted and
	// resolved by Stream.Save (CalculateMentions), so the object arrives fully tagged.
	object := service.JSONLD(session, stream)

	// RULE: A reply must reach the AUTHOR of the post it replies to, so they receive it (and a Reply
	// notification) even when they do not follow the replier — the common case for replies. We add
	// that author to the reply's `cc`, which the to/cc copy below carries onto the Create/Update
	// wrapper; Outbox.Publish then delivers to every addressee on top of the follower fan-out. This
	// mirrors how an Announce cc's the reacted-to author (see service.Response.reactionAudience).
	if authorURL := service.inReplyToAuthorURL(stream); authorURL != "" {
		cc, _ := object[vocab.PropertyCC].([]string)
		if !slices.Contains(cc, authorURL) {
			object[vocab.PropertyCC] = append(cc, authorURL)
		}
	}

	// Save the object to the ActivityStream cache
	if err := service.activityService.Save(streams.NewDocument(object)); err != nil {
		return derp.Wrap(err, location, "Saving object to ActivityStream cache", object)
	}

	// If this has not been published yet, then `Create` activity. Otherwise, `Update`
	activityType := iif(
		wasPublished,
		vocab.ActivityTypeUpdate,
		vocab.ActivityTypeCreate,
	)

	// Create the Activity to send to Followers.
	//
	// NOTE: a Create/Update is an ACTIVITY wrapping an OBJECT (the Stream). The activity has
	// no record of its own, so it carries NO `id` — the Outbox mints one on publish. The
	// wrapped `object` keeps its own object-id (stream.ActivityPubURL()); the activity must
	// not borrow it, or the two share an identity. See COLLECTIONS-REDESIGN.md D7.
	activity := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyType:      activityType,
		vocab.PropertyActor:     user.ActivityPubURL(),
		vocab.PropertyObject:    object,
		vocab.PropertyPublished: datetime.Now(),
	}

	if to, ok := object[vocab.PropertyTo]; ok {
		activity[vocab.PropertyTo] = to
	}

	if cc, ok := object[vocab.PropertyCC]; ok {
		activity[vocab.PropertyCC] = cc
	}

	// Publish to the User's outbox
	if err := service.publish_outbox_user(session, user, stream, activity); err != nil {
		return derp.Wrap(err, location, "Publishing to User's outbox")
	}

	// Publish to the parent Stream's outbox
	if err := service.publish_outbox_stream(session, stream, activity); err != nil {
		return derp.Wrap(err, location, "Publishing to parent Stream's outbox")
	}

	return nil
}

// publish_outbox_user publishes this stream to the User's outbox
func (service *Stream) publish_outbox_user(session data.Session, user *model.User, stream *model.Stream, activity mapof.Any) error {

	const location = "service.Stream.publish_outbox_user"

	// RULE: Do not allow empty Users
	if user == nil {
		return derp.Internal(location, "User cannot be nil")
	}

	// RULE: Do not allow "new" Users
	if user.IsNew() {
		return nil
	}

	// Try to publish via sendNotifications
	objectID := activity.GetString(vocab.PropertyID)
	objectType := activity.GetString(vocab.PropertyType)
	log.Trace().Str("location", location).Str("objectId", objectID).Str("type", objectType).Msg("Publishing to User's outbox")

	document := streams.NewDocument(activity)

	if err := service.outboxService.Publish(session, model.FollowerTypeUser, user.UserID, document, stream.DefaultAllow); err != nil {
		return derp.Wrap(err, location, "Publishing activity to user's outbox", activity)
	}

	// Done.
	return nil
}

// publish_outbox_stream publishes this Stream to the parent Stream's outbox
func (service *Stream) publish_outbox_stream(session data.Session, stream *model.Stream, activity mapof.Any) error {

	const location = "service.Stream.publish_outbox_stream"

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

	// Make a new "Announce/Boost" activity so that our encryption keys are correct.
	announce := mapof.Any{
		vocab.AtContext:      vocab.ContextTypeActivityStreams,
		vocab.PropertyType:   vocab.ActivityTypeAnnounce,
		vocab.PropertyActor:  service.ActivityPubURL(stream.ParentID),
		vocab.PropertyObject: activity,
	}

	document := streams.NewDocument(announce)

	// Try to publish via sendNotifications
	log.Trace().Str("id", stream.URL).Msg("Publishing to parent Stream's outbox")
	if err := service.outboxService.Publish(session, model.FollowerTypeStream, stream.ParentID, document, stream.DefaultAllow); err != nil {
		return derp.Wrap(err, location, "Publishing activity to parent Stream outbox", activity)
	}

	// Done.
	return nil
}

// inReplyToAuthorURL resolves the AUTHOR (attributedTo) of the post this Stream replies to, so a
// reply can be delivered to that author's inbox. Only actors have inboxes, so the author — not the
// parent object's URL — is the deliverable target of a reply. Returns "" when the Stream is not a
// reply, or the parent cannot be loaded, or it has no attributedTo. Mirrors the author resolution
// that service.Response.objectAuthorURL performs for reactions.
func (service *Stream) inReplyToAuthorURL(stream *model.Stream) string {

	const location = "service.Stream.inReplyToAuthorURL"

	// RULE: A non-reply (empty inReplyTo) has no parent author to address.
	if stream.InReplyTo == "" {
		return ""
	}

	// Load the parent using a client scoped to the reply's author (the same client CalcContext uses).
	client := service.activityService.UserClient(stream.AttributedTo.UserID)
	parent, err := client.Load(stream.InReplyTo)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading in-reply-to document to resolve its author", stream.InReplyTo))
		return ""
	}

	return parent.AttributedTo().ID()
}
