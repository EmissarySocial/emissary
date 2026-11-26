package service

import (
	"slices"

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

// Publish moves this Stream into the provided state, with the User as its author.  The save
// dates the Stream when the state puts it in the outbox for the first time.
// The save queues the outbox task, which is what sends to followers and syndication targets.
func (service *Stream) Publish(session data.Session, user *model.User, stream *model.Stream, stateID string, republish bool) error {

	const location = "service.Stream.Publish"

	// RULE: Set Author to the currently logged in user.
	stream.SetAttributedTo(user.PersonLink())

	// RULE: Set the new state ID
	stream.StateID.Set(stateID)

	// RULE: `republish` asks the outbox task to send every syndication target again,
	// even when followers are already up to date
	mode := outboxModeQueue

	if republish {
		mode = outboxModeRepublish
	}

	// Re-save the Stream with the updated values.
	if err := service.save(session, stream, "Publishing", mode); err != nil {
		return derp.Wrap(err, location, "Saving stream", stream.StreamID)
	}

	return nil
}

// publish_outbox sends the Stream's object to its author's Outbox, as the provided Create or Update
func (service *Stream) publish_outbox(session data.Session, user *model.User, stream *model.Stream, activityType string, object mapof.Any) error {

	const location = "service.Stream.publish_outbox"

	// Save the object to the ActivityStream cache
	if err := service.activityService.Save(streams.NewDocument(object)); err != nil {
		return derp.Wrap(err, location, "Saving object to ActivityStream cache", object)
	}

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

	// The activity reaches exactly the audience its object names
	copyAddressing(activity, object)

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

// addReplyAuthor adds the author of the post that the Stream replies to into the `cc` of its object
func (service *Stream) addReplyAuthor(stream *model.Stream, object mapof.Any) {

	// RULE: A reply must reach the AUTHOR of the post it replies to, so they receive it (and a Reply
	// notification) even when they do not follow the replier. JSONLD leaves this out because finding
	// the author can mean a fetch, and JSONLD also serves every page view.
	authorURL := service.inReplyToAuthorURL(stream)

	if authorURL == "" {
		return
	}

	if cc := object.GetSliceOfString(vocab.PropertyCC); !slices.Contains(cc, authorURL) {
		object[vocab.PropertyCC] = append(cc, authorURL)
	}
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

	// Publish through Outbox2, which stores the activity and queues its delivery
	objectID := activity.GetString(vocab.PropertyID)
	objectType := activity.GetString(vocab.PropertyType)
	log.Trace().Str("location", location).Str("objectId", objectID).Str("type", objectType).Msg("Publishing to User's outbox")

	document := streams.NewDocument(activity)

	if _, err := service.outbox2Service.Publish(session, model.ActorTypeUser, user.UserID, document, stream.DefaultAllow); err != nil {
		return derp.Wrap(err, location, "Publishing activity to user's outbox", "streamId: "+stream.StreamID.Hex())
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
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyType:      vocab.ActivityTypeAnnounce,
		vocab.PropertyActor:     service.ActivityPubURL(stream.ParentID),
		vocab.PropertyObject:    activity,
		vocab.PropertyPublished: datetime.Now(),
	}

	// RULE: The parent announces to the Stream's audience, as the parent sees it. A Stream names no
	// circles, so a circle-only child is announced to no one.
	copyAddressing(announce, service.parentAudience(stream))

	document := streams.NewDocument(announce)

	// Publish through Outbox2, which stores the activity and queues its delivery
	log.Trace().Str("id", stream.URL).Msg("Publishing to parent Stream's outbox")
	if _, err := service.outbox2Service.Publish(session, model.ActorTypeStream, stream.ParentID, document, stream.DefaultAllow); err != nil {
		return derp.Wrap(err, location, "Publishing activity to parent Stream outbox", "streamId: "+stream.StreamID.Hex())
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

// parentAudience returns the addressing of an activity that a Stream's parent sends about it
func (service *Stream) parentAudience(stream *model.Stream) mapof.Any {
	to, cc := service.Addressing(stream, model.ActorTypeStream, stream.ParentID)
	return mapof.Any{vocab.PropertyTo: to, vocab.PropertyCC: cc}
}
