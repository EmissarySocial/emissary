package service

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// These tests run the save half of Stream.Import against a real MongoDB, because the fingerprint
// is recorded by a conditional update.  The other half of Import fetches from the source server.

// exportedStream returns a Stream as an import decodes it from another server's export
func (test outboxSyncTest) exportedStream(t *testing.T, stateID string) model.Stream {

	t.Helper()

	original := test.newStream(stateID)
	original.IsOutbox.Set(stateID == "published")
	original.PublishDate = 1_700_000_000

	encoded, err := json.Marshal(original)
	require.NoError(t, err)

	result := model.NewStream()
	require.NoError(t, json.Unmarshal(encoded, &result))
	result.CreateDate = 0

	return result
}

// TestStream_SaveImported_SendsNothing confirms that an imported Stream in an outbox State is
// stored in the outbox, with its date, and that nothing is queued for followers
func TestStream_SaveImported_SendsNothing(t *testing.T) {

	test := newOutboxSyncTest(t)
	stream := test.exportedStream(t, "published")

	require.NoError(t, test.service.saveImported(test.session, &stream))
	require.Empty(t, test.tasks.Drain(), "an import queues no task at all")

	stored := storedStream(t, test.service, test.session, stream.StreamID)
	require.True(t, stored.IsOutbox.Value())
	require.True(t, stored.IsPublished())
	require.Equal(t, int64(1_700_000_000), stored.PublishDate, "the original date is kept")
	require.NotEmpty(t, stored.Fingerprint.Value())
	require.NotEqual(t, model.StreamFingerprintPending, stored.Fingerprint.Value())
}

// TestStream_SaveImported_LaterSaves confirms what followers are sent after an import: nothing
// for a save that changes nothing, an Update for an edit, and a Delete for a delete
func TestStream_SaveImported_LaterSaves(t *testing.T) {

	test := newOutboxSyncTest(t)
	stream := test.exportedStream(t, "published")
	require.NoError(t, test.service.saveImported(test.session, &stream))

	// The same Stream, saved again as another request would load it
	loaded := storedStream(t, test.service, test.session, stream.StreamID)
	require.NoError(t, test.service.Save(test.session, &loaded, "saved again"))
	require.Empty(t, test.sync(t), "followers already hold this copy")

	// An edit is an Update, never a late Create
	loaded = storedStream(t, test.service, test.session, stream.StreamID)
	loaded.Label = "No Fate"
	require.NoError(t, test.service.Save(test.session, &loaded, "edited"))
	require.Equal(t, []string{vocab.ActivityTypeUpdate}, test.sync(t))

	// A delete is still sent
	loaded = storedStream(t, test.service, test.session, stream.StreamID)
	require.NoError(t, test.service.Delete(test.session, &loaded, "deleted"))
	require.Equal(t, []string{vocab.ActivityTypeDelete}, test.sync(t))
}

// TestStream_SaveImported_Draft confirms that an imported Stream outside the outbox gets no
// fingerprint, so that publishing it later sends a Create
func TestStream_SaveImported_Draft(t *testing.T) {

	test := newOutboxSyncTest(t)
	stream := test.exportedStream(t, "default")
	stream.PublishDate = math.MaxInt64

	require.NoError(t, test.service.saveImported(test.session, &stream))
	require.Empty(t, test.tasks.Drain())

	stored := storedStream(t, test.service, test.session, stream.StreamID)
	require.False(t, stored.IsOutbox.Value())
	require.Empty(t, stored.Fingerprint.Value())
	require.Zero(t, stored.PublishDate, "an older export's sentinel date is cleared")

	stored.StateID.Set("published")
	require.NoError(t, test.service.Save(test.session, &stored, "published"))
	require.Equal(t, []string{vocab.ActivityTypeCreate}, test.sync(t))
}

// TestStream_SaveImported_OlderExport confirms that an export with no "isOutbox" property takes
// the flag from its State, and is treated the same way
func TestStream_SaveImported_OlderExport(t *testing.T) {

	test := newOutboxSyncTest(t)
	stream := test.newStream("published")
	stream.PublishDate = 1_700_000_000

	require.NoError(t, test.service.saveImported(test.session, &stream))
	require.Empty(t, test.tasks.Drain())

	stored := storedStream(t, test.service, test.session, stream.StreamID)
	require.True(t, stored.IsOutbox.Value())
	require.NotEmpty(t, stored.Fingerprint.Value())
}

// TestStream_SaveImported_Reply confirms that a reply is imported without loading the post it
// answers, which may no longer exist
func TestStream_SaveImported_Reply(t *testing.T) {

	test := newOutboxSyncTest(t)
	stream := test.newPublicStream("published")
	stream.InReplyTo = "https://gone.example.test/a-post-that-no-longer-exists"

	// An ordinary save has to load the parent, and fails
	ordinary := test.newPublicStream("published")
	ordinary.InReplyTo = stream.InReplyTo
	require.Error(t, test.service.Save(test.session, &ordinary, "ordinary reply"))

	require.NoError(t, test.service.saveImported(test.session, &stream))
	require.Empty(t, stream.Context, "an import keeps the context it arrived with, and makes none")
	require.NotEmpty(t, storedStream(t, test.service, test.session, stream.StreamID).Fingerprint.Value())
}
