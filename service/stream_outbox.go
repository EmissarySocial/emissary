package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamOutboxTaskName names the queue task that brings a Stream's followers up to date
const StreamOutboxTaskName = "Stream-Outbox"

// outboxMode says what a save does about the Stream's outbox task
type outboxMode int

const (
	outboxModeQueue     outboxMode = iota // Queue the task, which is what every ordinary save does
	outboxModeSkip                        // Queue nothing, for a change that followers never receive
	outboxModeRepublish                   // Queue the task, and send every syndication target again
)

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
	if stream.PublishDate == 0 {
		stream.PublishDate = time.Now().Unix()
	}
}

// queueOutbox queues the task that decides, after commit, what this Stream's followers and
// syndication targets should be sent
func (service *Stream) queueOutbox(session data.Session, stream *model.Stream, republish bool) {

	// RULE: Only a Stream that is, was, or is still held to be in the outbox can need a send
	if stream.IsOutbox.IsFalse() && !stream.IsOutbox.Original() && stream.Fingerprint.IsZero() {
		return
	}

	args := mapof.Any{
		"streamId": stream.StreamID.Hex(),
	}

	// RULE: The changes to the syndication targets exist only in this copy of the Stream,
	// so the task is told what they were
	if added := stream.Syndication.Added(); len(added) > 0 {
		args["syndicationAdded"] = added
	}

	if deleted := stream.Syndication.Deleted(); len(deleted) > 0 {
		args["syndicationDeleted"] = deleted
	}

	if republish {
		args["republish"] = true
	}

	postcommit.Publish(session, service.queue, StreamOutboxTaskName, args)
}

// joinConversation gives a Stream that is entering the outbox its context, and records it as a
// reply in the Replies collection of a local parent
func (service *Stream) joinConversation(session data.Session, stream *model.Stream) error {

	const location = "service.Stream.joinConversation"

	// RULE: Only the save that moves a Stream into the outbox does this
	if stream.IsOutbox.IsFalse() || stream.IsOutbox.Original() {
		return nil
	}

	// Guarantee this Stream has a context collection
	if err := service.CalcContext(session, stream); err != nil {
		return derp.Wrap(err, location, "Calculating context", stream.StreamID)
	}

	// A parent with closed replies simply leaves it out, and the save still goes ahead
	if err := service.AddReply(session, stream.InReplyTo, stream.ActivityPubURL()); err != nil {
		return derp.Wrap(err, location, "Adding reply to parent's collection", stream.StreamID)
	}

	return nil
}

// syndicationPlan returns the syndication targets that are sent a Create, an Update, and a Delete.
// `action` is the activity that followers were sent, or "" when they were sent nothing, and
// `wanted` is TRUE when the author wants the Stream in the outbox.
func syndicationPlan(action string, wanted bool, current []string, changes OutboxChanges) (created []string, updated []string, deleted []string) {

	switch {

	// Followers were told the Stream is gone, so every target that holds it is told too
	case action == vocab.ActivityTypeDelete:
		return nil, nil, append(heldTargets(current, changes), changes.SyndicationDeleted...)

	// A Stream outside the outbox, which followers do not hold, was never syndicated
	case !wanted:
		return nil, nil, nil

	// Followers were sent the Stream for the first time, so every target is too
	case action == vocab.ActivityTypeCreate:
		return slices.Clone(current), nil, nil
	}

	// A target that the same save added has received nothing yet, so it is sent a Create
	added := slices.DeleteFunc(slices.Clone(current), func(target string) bool {
		return !slices.Contains(changes.SyndicationAdded, target)
	})

	// Followers were sent a new copy, or every target is to be sent again
	if (action == vocab.ActivityTypeUpdate) || changes.Republish {
		return added, heldTargets(current, changes), changes.SyndicationDeleted
	}

	// Followers are up to date, and so is every target that did not change
	return added, nil, changes.SyndicationDeleted
}

// heldTargets returns the current syndication targets that held the Stream before the save
// that queued this task, which leaves out the ones that save added
func heldTargets(current []string, changes OutboxChanges) []string {
	return slices.DeleteFunc(slices.Clone(current), func(target string) bool {
		return slices.Contains(changes.SyndicationAdded, target)
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
// records what they now hold, and tells its syndication targets the same.  `changes` is what
// the save that queued this task passed on.
func (service *Stream) SyncOutbox(session data.Session, streamID primitive.ObjectID, changes OutboxChanges) error {

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
		return service.syncOutboxRemove(session, &stream, changes)
	}

	return service.syncOutboxSend(session, &author, &stream, changes)
}

// syncOutboxSend sends a Create or an Update when followers do not hold the Stream as it is now
func (service *Stream) syncOutboxSend(session data.Session, author *model.User, stream *model.Stream, changes OutboxChanges) error {

	const location = "service.Stream.syncOutboxSend"

	object, fingerprint, err := service.outboxObject(session, stream)

	if err != nil {
		return derp.Wrap(err, location, "Building object", stream.StreamID)
	}

	activityType := outboxAction(true, stream.Fingerprint.Value(), fingerprint)

	// Followers are up to date.  Syndication targets may still have changed, or be sent again.
	if activityType == "" {
		return service.syncOutboxSyndication(session, stream, "", true, changes)
	}

	// Address the object to the author of the post it replies to.  That lookup can mean a
	// fetch, so it stays out of the fingerprint, where a failed fetch would read as an edit.
	service.addReplyAuthor(stream, object)

	// RULE: Claim the send before making it.  Another server that ran this task for the same
	// Stream has already sent, and moved the fingerprint away from the value read here.
	claimed, err := service.claimFingerprint(session, stream, fingerprint)

	if err != nil {
		return derp.Wrap(err, location, "Recording fingerprint", stream.StreamID)
	}

	if !claimed {
		return nil
	}

	if err := service.publish_outbox(session, author, stream, activityType, object); err != nil {
		return derp.Wrap(err, location, "Publishing to outbox", stream.StreamID)
	}

	// Engage
	return service.syncOutboxSyndication(session, stream, activityType, true, changes)
}

// syncOutboxRemove sends a Delete when followers still hold a Stream that has left the outbox
func (service *Stream) syncOutboxRemove(session data.Session, stream *model.Stream, changes OutboxChanges) error {

	const location = "service.Stream.syncOutboxRemove"

	if outboxAction(false, stream.Fingerprint.Value(), "") == "" {
		return nil
	}

	// RULE: Claim the Delete before sending it, as a Create or an Update is claimed
	claimed, err := service.claimFingerprint(session, stream, "")

	if err != nil {
		return derp.Wrap(err, location, "Clearing fingerprint", stream.StreamID)
	}

	if !claimed {
		return nil
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
	return service.syncOutboxSyndication(session, stream, vocab.ActivityTypeDelete, false, changes)
}

// syncOutboxSyndication tells a Stream's syndication targets what its followers were just sent
func (service *Stream) syncOutboxSyndication(session data.Session, stream *model.Stream, action string, wanted bool, changes OutboxChanges) error {

	const location = "service.Stream.syncOutboxSyndication"

	created, updated, deleted := syndicationPlan(action, wanted, stream.Syndication.Value(), changes)

	if err := service.sendSyndicationMessages(session, stream, created, updated, deleted); err != nil {
		return derp.Wrap(err, location, "Sending syndication messages", stream.StreamID)
	}

	return nil
}

// claimFingerprint records what followers now hold of a Stream, and returns FALSE when the stored
// fingerprint is no longer the one this copy of the Stream was loaded with
func (service *Stream) claimFingerprint(session data.Session, stream *model.Stream, fingerprint string) (bool, error) {

	const location = "service.Stream.claimFingerprint"

	// RULE: Never through Stream.Save, which would write the whole Stream over an edit made
	// meanwhile, and queue this task again.  The claim shares the task's transaction with the
	// send that follows it, so a send that fails takes its claim back.
	claimed, err := queries.ClaimStreamFingerprint(session, stream.StreamID, stream.Fingerprint.Value(), fingerprint)

	if err != nil {
		return false, derp.Wrap(err, location, "Updating Stream", stream.StreamID)
	}

	return claimed, nil
}

// outboxObject returns the object that followers are sent for a Stream, and its fingerprint
func (service *Stream) outboxObject(session data.Session, stream *model.Stream) (mapof.Any, string, error) {

	const location = "service.Stream.outboxObject"

	// RULE: The Template supplies the hashtag URLs and the social mapping, so nothing is built
	// without it.  JSONLD would report the error and return an object with neither.
	if _, err := service.templateService.Load(stream.TemplateID); err != nil {
		return nil, "", derp.Wrap(err, location, "Loading Template", stream.TemplateID)
	}

	// RULE: Fingerprint the whole object, social mapping included, so that a change to any
	// custom field the mapping reads is sent to followers
	object := service.JSONLD(session, stream)
	fingerprint, err := outboxFingerprint(object)

	if err != nil {
		return nil, "", derp.Wrap(err, location, "Calculating fingerprint", stream.StreamID)
	}

	return object, fingerprint, nil
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
