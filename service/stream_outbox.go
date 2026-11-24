package service

import (
	"math"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
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
