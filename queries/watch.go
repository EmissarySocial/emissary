package queries

/******************************************
 * Supervised Change Streams
 *
 * A MongoDB change stream can end without an error when the
 * server closes it, so every watcher in this package runs
 * inside changeWatcher, which reopens it until canceled.
 ******************************************/

import (
	"context"
	"errors"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// watchRetryMinimum is the shortest pause before a closed change stream is reopened
const watchRetryMinimum = 1 * time.Second

// watchRetryMaximum caps the backoff between attempts to reopen a change stream
const watchRetryMaximum = 60 * time.Second

// mongoErrorNotReplicaSet is the MongoDB error code for a change stream opened on a standalone server
const mongoErrorNotReplicaSet = 40573

// changeWatcher describes one supervised change stream on a single collection
type changeWatcher struct {
	collection   string                                                        // name of the collection to watch
	fullDocument options.FullDocument                                          // UpdateLookup makes update events carry the document, too
	onOpen       func(ctx context.Context, collection *mongo.Collection) error // resynchronizes after every open (optional)
	onDocument   func(ctx context.Context, document bson.Raw)                  // receives the document carried by each change event
}

// run supervises the change stream until ctx is canceled, reopening it whenever it ends
func (watcher changeWatcher) run(ctx context.Context, server data.Server) {

	const location = "queries.changeWatcher.run"

	// Open a session that lasts as long as the context
	session, err := server.Session(ctx)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Opening database session", watcher.collection))
		return
	}

	defer session.Close()

	// RULE: Only a MongoDB collection has change streams
	collection := mongoCollection(session.Collection(watcher.collection))

	if collection == nil {
		return
	}

	// The resume token carries over from each stream to the next
	var resumeToken bson.Raw

	// `failures` counts CONSECUTIVE failed attempts, and resets once a stream delivers an event
	for failures := 0; ; {

		progressed, err := watcher.runOnce(ctx, collection, &resumeToken)

		// RULE: Cancellation is the only way out of this loop
		if ctx.Err() != nil {
			return
		}

		// RULE: A standalone server can never open a change stream, so stop trying
		if isNotReplicaSet(err) {
			return
		}

		// A stream that delivered events was healthy, so the backoff starts over
		if progressed {
			failures = 0
		}

		// Report why the stream ended.  A server-side close arrives with no error at all.
		if err != nil {
			derp.Report(derp.Wrap(err, location, "Change stream failed. Reopening.", watcher.collection))
		} else {
			log.Warn().Str("loc", location).Str("collection", watcher.collection).Msg("Change stream was closed by the server. Reopening.")
		}

		failures++

		// Wait before reopening, but stay cancelable
		select {
		case <-ctx.Done():
			return
		case <-time.After(watchRetryDelay(failures)):
		}
	}
}

// runOnce opens one change stream and delivers its events until it ends.  It reports whether any
// event arrived, and the error that ended the stream (nil when the server closed it).
func (watcher changeWatcher) runOnce(ctx context.Context, collection *mongo.Collection, resumeToken *bson.Raw) (bool, error) {

	const location = "queries.changeWatcher.runOnce"

	// Resume after the last event seen, so a reopened stream misses nothing.  StartAfter (unlike
	// ResumeAfter) also resumes past an `invalidate`, which is how a dropped collection ends a stream.
	streamOptions := options.ChangeStream()

	if watcher.fullDocument != "" {
		streamOptions.SetFullDocument(watcher.fullDocument)
	}

	if *resumeToken != nil {
		streamOptions.SetStartAfter(*resumeToken)
	}

	// Open the stream
	changeStream, err := collection.Watch(ctx, mongo.Pipeline{}, streamOptions)

	if err != nil {

		// RULE: Drop the token after a failed open.  A token the server refuses would fail every retry.
		*resumeToken = nil
		return false, derp.Wrap(err, location, "Opening change stream", watcher.collection)
	}

	// RULE: Guard the driver's (stream, error) contract, because this loop must never panic
	if changeStream == nil {
		return false, derp.Internal(location, "MongoDB returned no change stream, and no error", watcher.collection)
	}

	// Close the stream however this attempt ends.  WithoutCancel, because ctx may already be
	// canceled, and closing still has to reach the server.  A failed close changes nothing here.
	defer func() {
		_ = changeStream.Close(context.WithoutCancel(ctx))
	}()

	// Remember where this stream ended, so the next one resumes there
	defer func() {
		if token := changeStream.ResumeToken(); token != nil {
			*resumeToken = token
		}
	}()

	// Resynchronize, because events may have been missed while no stream was open
	if watcher.onOpen != nil {
		if err := watcher.onOpen(ctx, collection); err != nil {
			return false, derp.Wrap(err, location, "Resynchronizing after opening the change stream", watcher.collection)
		}
	}

	// Deliver every event that carries a document
	progressed := false

	for changeStream.Next(ctx) {

		progressed = true

		if document, ok := fullDocument(changeStream.Current); ok {
			watcher.onDocument(ctx, document)
		}
	}

	// WrapIF, because a stream the server closed ends with no error, and derp.Wrap never returns nil
	return progressed, derp.WrapIF(changeStream.Err(), location, "Reading change stream", watcher.collection)
}

// fullDocument returns the document carried by a change event, if the event has one
func fullDocument(event bson.Raw) (bson.Raw, bool) {

	value, err := event.LookupErr("fullDocument")

	if err != nil {
		return nil, false
	}

	return value.DocumentOK()
}

// isNotReplicaSet returns TRUE if the error means the server is a standalone node, not a replica set
func isNotReplicaSet(err error) bool {

	var serverError mongo.ServerError

	if errors.As(err, &serverError) {
		return serverError.HasErrorCode(mongoErrorNotReplicaSet)
	}

	return false
}

// watchRetryDelay returns the pause before the Nth consecutive attempt to reopen a change stream:
// 1s, 2s, 4s, 8s ... capped at watchRetryMaximum.
func watchRetryDelay(failures int) time.Duration {

	result := watchRetryMinimum

	for index := 1; index < failures; index++ {

		result *= 2

		if result >= watchRetryMaximum {
			return watchRetryMaximum
		}
	}

	return result
}
