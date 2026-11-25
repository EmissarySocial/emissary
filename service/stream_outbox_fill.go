package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamFillFingerprintsTaskName names the queue task that replaces placeholder fingerprints with real ones
const StreamFillFingerprintsTaskName = "Stream-FillFingerprints"

// fillFingerprintsPageSize is the number of Streams that one run of the task reads
const fillFingerprintsPageSize = 100

// FillFingerprints replaces the placeholder fingerprint on one page of Streams, reading those
// after `afterID`.  It sends nothing.  It returns the last StreamID it read, and TRUE when
// no page remains.
func (service *Stream) FillFingerprints(session data.Session, afterID primitive.ObjectID) (primitive.ObjectID, bool, error) {

	const location = "service.Stream.FillFingerprints"

	// NILCHECK: Session cannot be nil
	if session == nil {
		return afterID, false, derp.Internal(location, "Session cannot be nil. This should never happen.")
	}

	// Read one page, in ID order, so that a Stream this run cannot fill does not stall the next
	criteria := exp.Equal("fingerprint", model.StreamFingerprintPending).And(exp.GreaterThan("_id", afterID))
	streams, err := service.Query(session, criteria, option.SortAsc("_id"), option.MaxRows(fillFingerprintsPageSize))

	if err != nil {
		return afterID, false, derp.Wrap(err, location, "Loading Streams", afterID)
	}

	for index := range streams {

		stream := &streams[index]
		afterID = stream.StreamID

		// A Stream that cannot be built keeps its placeholder, which the outbox task reads as "differs"
		_, fingerprint, err := service.outboxObject(session, stream)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Building object", stream.StreamID))
			continue
		}

		// RULE: The claim matches the placeholder, so a fingerprint that the outbox task has
		// recorded since this page was read is never written over
		if _, err := service.claimFingerprint(session, stream, fingerprint); err != nil {
			return afterID, false, derp.Wrap(err, location, "Recording fingerprint", stream.StreamID)
		}
	}

	// A short page is the last one
	return afterID, len(streams) < fillFingerprintsPageSize, nil
}
