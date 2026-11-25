package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamOutboxTaskName names the queue task that brings a Stream's followers up to date
const StreamOutboxTaskName = "Stream-Outbox"

// calcOutbox copies the "outbox" flag of the Stream's State onto the Stream when its State has changed
func calcOutbox(template *model.Template, stream *model.Stream, wasNew bool) {

	// RULE: A new Stream counts as a change of State, and followers hold nothing of it
	if wasNew {
		stream.Fingerprint.Clear()
		applyOutboxState(template, stream)
		return
	}

	// RULE: A save never changes the fingerprint.  Only the outbox task's claim writes it.
	stream.Fingerprint.Set(stream.Fingerprint.Original())

	// RULE: The flag is copied only when the State changes, so editing a Template moves no existing Stream
	if stream.StateID.NotChanged() {
		stream.IsOutbox.Set(stream.IsOutbox.Original())
		return
	}

	applyOutboxState(template, stream)
}

// applyOutboxState sets the Stream's "outbox" flag from its State in the Template, and dates the
// Stream's first entry into the outbox
func applyOutboxState(template *model.Template, stream *model.Stream) {

	// A State that the Template does not define sends nothing
	state, _ := template.State(stream.StateID.Value())
	stream.IsOutbox.Set(state.IsOutbox)

	if stream.IsOutbox.IsFalse() {
		return
	}

	// RULE: The publish date is set once, so a backdated or republished Stream keeps its date
	if (stream.PublishDate == 0) || (stream.PublishDate == math.MaxInt64) {
		stream.PublishDate = time.Now().Unix()
	}
}

// queueOutbox queues the task that decides, after commit, what this Stream's followers should be sent
func (service *Stream) queueOutbox(session data.Session, stream *model.Stream) {

	// RULE: Only a Stream that is, was, or is still held to be in the outbox can need a send
	if stream.IsOutbox.IsFalse() && !stream.IsOutbox.Original() && stream.Fingerprint.IsZero() {
		return
	}

	postcommit.Publish(session, service.queue, StreamOutboxTaskName, mapof.Any{
		"streamId": stream.StreamID.Hex(),
	})
}

// outboxAction returns the type of activity that brings followers up to date, or "" when they already are.
// `wanted` is TRUE when the author wants the Stream in the outbox, `held` is the fingerprint of
// what followers hold, and `current` is the fingerprint of the Stream as it is now.
func outboxAction(wanted bool, held string, current string) string {

	// Followers hold something that the author no longer wants them to have
	if !wanted {
		if held == "" {
			return ""
		}
		return vocab.ActivityTypeDelete
	}

	// Followers hold nothing yet
	if held == "" {
		return vocab.ActivityTypeCreate
	}

	// Followers hold an older copy
	if held != current {
		return vocab.ActivityTypeUpdate
	}

	return ""
}

// SyncOutbox sends the Create, Update, or Delete that brings a Stream's followers up to date,
// and records what they now hold.  It sends nothing when they are already up to date.
func (service *Stream) SyncOutbox(session data.Session, streamID primitive.ObjectID) error {

	const location = "service.Stream.SyncOutbox"

	// NILCHECK: Session cannot be nil
	if session == nil {
		return derp.Internal(location, "Session cannot be nil. This should never happen.")
	}

	// Load the Stream, deleted or not.  A Stream that is gone from the database has nothing to send.
	stream := model.NewStream()

	if err := service.collection(session).Load(exp.Equal("_id", streamID), &stream); err != nil {
		if derp.IsNotFound(err) {
			return nil
		}
		return derp.Wrap(err, location, "Loading Stream", streamID)
	}

	// RULE: A Stream with no author was never sent to a User's followers
	authorID := stream.AttributedTo.UserID

	if authorID.IsZero() {
		return nil
	}

	// RULE: An author who no longer exists sends nothing.  Deleting a User clears their outbox
	// first, so that it does not send a Delete for every one of their Streams.
	author := model.NewUser()

	if err := service.userService.LoadByID(session, authorID, &author); err != nil {
		if derp.IsNotFound(err) {
			return nil
		}
		return derp.Wrap(err, location, "Loading author", authorID)
	}

	// The author wants a Stream in the outbox when it is flagged and not deleted
	if stream.IsOutbox.IsFalse() || stream.IsDeleted() {
		return service.syncOutboxRemove(session, &stream)
	}

	return service.syncOutboxSend(session, &author, &stream)
}

// syncOutboxSend sends a Create or an Update when followers do not hold the Stream as it is now
func (service *Stream) syncOutboxSend(session data.Session, author *model.User, stream *model.Stream) error {

	const location = "service.Stream.syncOutboxSend"

	// RULE: The Template supplies the hashtag URLs and the social mapping, so nothing is sent
	// without it.  JSONLD would report the error and return an object with neither.
	if _, err := service.templateService.Load(stream.TemplateID); err != nil {
		return derp.Wrap(err, location, "Loading Template", stream.TemplateID)
	}

	// RULE: Fingerprint the whole object, social mapping included, so that a change to any
	// custom field the mapping reads is sent to followers
	object := service.JSONLD(session, stream)
	fingerprint, err := outboxFingerprint(object)

	if err != nil {
		return derp.Wrap(err, location, "Calculating fingerprint", stream.StreamID)
	}

	activityType := outboxAction(true, stream.Fingerprint.Value(), fingerprint)

	if activityType == "" {
		return nil
	}

	// Address the object to the author of the post it replies to.  That lookup can mean a
	// fetch, so it stays out of the fingerprint, where a failed fetch would read as an edit.
	service.addReplyAuthor(stream, object)

	// Record what followers hold.  This shares the task's transaction with the send below.
	if err := service.setFingerprint(session, stream.StreamID, fingerprint); err != nil {
		return derp.Wrap(err, location, "Recording fingerprint", stream.StreamID)
	}

	if err := service.publish_outbox(session, author, stream, activityType, object); err != nil {
		return derp.Wrap(err, location, "Publishing to outbox", stream.StreamID)
	}

	// Engage
	return nil
}

// syncOutboxRemove sends a Delete when followers still hold a Stream that has left the outbox
func (service *Stream) syncOutboxRemove(session data.Session, stream *model.Stream) error {

	const location = "service.Stream.syncOutboxRemove"

	if outboxAction(false, stream.Fingerprint.Value(), "") == "" {
		return nil
	}

	// Record that followers hold nothing.  This shares the task's transaction with the sends below.
	if err := service.setFingerprint(session, stream.StreamID, ""); err != nil {
		return derp.Wrap(err, location, "Clearing fingerprint", stream.StreamID)
	}

	// Send the Delete to the author's followers
	if err := service.unpublish_outbox_user(session, stream.AttributedTo.UserID, stream); err != nil {
		return derp.Wrap(err, location, "Unpublishing from the User's outbox", stream.StreamID)
	}

	// Send the Delete to the parent Stream's followers
	if err := service.unpublish_outbox_stream(session, stream); err != nil {
		return derp.Wrap(err, location, "Unpublishing from the parent Stream's outbox", stream.StreamID)
	}

	// Gone with the wind
	return nil
}

// setFingerprint records what followers hold of a Stream, without saving the rest of it
func (service *Stream) setFingerprint(session data.Session, streamID primitive.ObjectID, fingerprint string) error {

	const location = "service.Stream.setFingerprint"

	// RULE: Never through Stream.Save, which would write the whole Stream over an edit
	// made meanwhile, and queue this task again
	update := bson.M{"$set": bson.M{"fingerprint": fingerprint}}

	if fingerprint == "" {
		update = bson.M{"$unset": bson.M{"fingerprint": ""}}
	}

	if err := queries.RawUpdate(session.Context(), service.collection(session), exp.Equal("_id", streamID), update); err != nil {
		return derp.Wrap(err, location, "Updating Stream", streamID)
	}

	return nil
}

// outboxFingerprint returns a hash of an object's JSON-LD, which is equal for equal objects
func outboxFingerprint(object mapof.Any) (string, error) {

	const location = "service.outboxFingerprint"

	// json.Marshal writes map keys in sorted order, so equal objects encode identically
	encoded, err := json.Marshal(object)

	if err != nil {
		return "", derp.Wrap(err, location, "Marshalling object")
	}

	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
