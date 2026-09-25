package consumer

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SyncStreamSource copies a remote source's content into the Stream that it populates
func SyncStreamSource(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.SyncStreamSource"

	streamSourceID, err := primitive.ObjectIDFromHex(args.GetString("streamSourceId"))

	if err != nil {
		// No retry can repair a malformed identifier
		return queue.Failure(derp.Wrap(err, location, "Invalid 'streamSourceId' argument", args))
	}

	// Load the StreamSource record
	streamSourceService := factory.StreamSource()
	streamSource := model.NewStreamSource()

	if err := streamSourceService.LoadByID(session, streamSourceID, &streamSource); err != nil {

		// A record deleted between the ping and this run has nothing left to synchronize
		if derp.IsNotFound(err) {
			return queue.Success()
		}

		return queue.Error(derp.Wrap(err, location, "Loading StreamSource", streamSourceID))
	}

	// Read the source, and push what it holds into the Stream
	if err := streamSourceService.Sync(session.Context(), session, &streamSource); err != nil {

		// requeue files the author's mistakes as permanent failures, and Emissary's as retries
		return requeue(derp.Wrap(err, location, "Synchronizing StreamSource", streamSourceID))
	}

	// Synced and delivered
	return queue.Success()
}

/******************************************
 * Lifecycle Hooks
 *
 * These run OUTSIDE the task's own transaction,
 * which has already committed or rolled back by
 * the time a hook fires.  That is what lets a
 * FAILURE status outlive the attempt that
 * produced it -- see AGENTS.md.
 ******************************************/

// syncStreamSourceSucceeded marks a record as synchronized
func syncStreamSourceSucceeded(serverFactory ServerFactory, args mapof.Any) error {

	return withStreamSource(serverFactory, args, func(streamSourceService *service.StreamSource, session data.Session, streamSource *model.StreamSource) error {
		return streamSourceService.SetStatusSuccess(session, streamSource)
	})
}

// syncStreamSourceRetrying records why an attempt failed, while another attempt is still queued
func syncStreamSourceRetrying(serverFactory ServerFactory, args mapof.Any, err error) error {

	return withStreamSource(serverFactory, args, func(streamSourceService *service.StreamSource, session data.Session, streamSource *model.StreamSource) error {
		return streamSourceService.SetStatusMessage(session, streamSource, derp.Message(err))
	})
}

// syncStreamSourceFailed marks a record as failed, once the queue has stopped retrying it
func syncStreamSourceFailed(serverFactory ServerFactory, args mapof.Any, err error) error {

	return withStreamSource(serverFactory, args, func(streamSourceService *service.StreamSource, session data.Session, streamSource *model.StreamSource) error {
		return streamSourceService.SetStatusFailure(session, streamSource, derp.Message(err))
	})
}

// withStreamSource loads the StreamSource record that a task names, and hands it to a status
// writer.  It is unexported, and it stays here rather than in wrappers.go, because the wrappers
// there are reusable dispatch scaffolding that returns a queue.Result; this serves one task's
// hooks and opens a plain session instead of a transaction.
func withStreamSource(serverFactory ServerFactory, args mapof.Any, handler func(*service.StreamSource, data.Session, *model.StreamSource) error) error {

	const location = "consumer.withStreamSource"

	// Resolve the Domain that owns this record
	factory, err := serverFactory.ByHostname(getHostnameFromArgs(args))

	if err != nil {
		return derp.Wrap(err, location, "Unrecognized hostname", args)
	}

	streamSourceID, err := primitive.ObjectIDFromHex(args.GetString("streamSourceId"))

	if err != nil {
		return derp.Wrap(err, location, "Invalid 'streamSourceId' argument", args)
	}

	// A plain session, deliberately NOT a transaction: the task's own transaction has already
	// settled, and a status written inside a failed one would roll back with it
	session, err := factory.Server().Session(context.Background())

	if err != nil {
		return derp.Wrap(err, location, "Opening database session")
	}

	defer session.Close()

	streamSourceService := factory.StreamSource()
	streamSource := model.NewStreamSource()

	if err := streamSourceService.LoadByID(session, streamSourceID, &streamSource); err != nil {

		// A record deleted while its task was running has no status left to report
		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading StreamSource", streamSourceID)
	}

	return handler(streamSourceService, session, &streamSource)
}
