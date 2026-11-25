package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests run the fingerprint fill against a real MongoDB, because the fill replaces each
// placeholder with a conditional update.

// setFingerprint stores a fingerprint on a Stream, as the upgrade or the outbox task would
func (test outboxSyncTest) setFingerprint(t *testing.T, streamID primitive.ObjectID, fingerprint string) {

	t.Helper()

	require.NoError(t, queries.RawUpdate(t.Context(), test.session.Collection("Stream"), exp.Equal("_id", streamID), bson.M{
		"$set": bson.M{"fingerprint": fingerprint},
	}))
}

// savePending stores a Stream in the provided State that followers received before fingerprints were recorded
func (test outboxSyncTest) savePending(t *testing.T, stateID string) model.Stream {

	t.Helper()

	stream := test.newStream(stateID)
	require.NoError(t, test.service.Save(test.session, &stream, "test setup"))
	test.setFingerprint(t, stream.StreamID, model.StreamFingerprintPending)
	test.tasks.Drain()

	return stream
}

// TestStream_FillFingerprints confirms that the fill replaces each placeholder with the fingerprint
// of the Stream as it is, sends nothing, and leaves every other Stream alone
func TestStream_FillFingerprints(t *testing.T) {

	test := newOutboxSyncTest(t)

	pending := test.savePending(t, "published")

	// A Stream that the outbox task has already recorded
	recorded := test.newStream("published")
	require.NoError(t, test.service.Save(test.session, &recorded, "test setup"))
	test.setFingerprint(t, recorded.StreamID, "recorded")

	// A Stream that followers never received
	draft := test.newStream("default")
	require.NoError(t, test.service.Save(test.session, &draft, "test setup"))

	// A deleted Stream, whose Delete is the outbox task's to send
	deleted := test.savePending(t, "published")
	deleted = storedStream(t, test.service, test.session, deleted.StreamID)
	require.NoError(t, test.session.Collection("Stream").Delete(&deleted, "test setup"))
	test.tasks.Drain()

	lastID, done, err := test.service.FillFingerprints(test.session, primitive.NilObjectID)
	require.NoError(t, err)
	require.True(t, done)
	require.Equal(t, pending.StreamID, lastID)
	require.Empty(t, test.sync(t), "the fill sends nothing")

	// The placeholder is now the fingerprint that the outbox task would calculate
	loaded := storedStream(t, test.service, test.session, pending.StreamID)
	_, expected, err := test.service.outboxObject(test.session, &loaded)
	require.NoError(t, err)
	require.Equal(t, expected, test.fingerprint(t, pending.StreamID))

	require.Equal(t, "recorded", test.fingerprint(t, recorded.StreamID))
	require.Empty(t, test.fingerprint(t, draft.StreamID))
	require.Equal(t, model.StreamFingerprintPending, test.fingerprint(t, deleted.StreamID))

	// A filled Stream is up to date, so the outbox task sends nothing for it
	require.NoError(t, test.service.SyncOutbox(test.session, pending.StreamID, OutboxChanges{}))
	require.Empty(t, test.sync(t))

	// A second fill finds nothing to do
	_, done, err = test.service.FillFingerprints(test.session, primitive.NilObjectID)
	require.NoError(t, err)
	require.True(t, done)
}

// TestStream_FillFingerprints_Pages confirms that the fill reads one page at a time, and that
// each run picks up where the last one stopped
func TestStream_FillFingerprints_Pages(t *testing.T) {

	test := newOutboxSyncTest(t)

	// One Stream more than a page holds, stored directly because only the fill is under test
	for range fillFingerprintsPageSize + 1 {
		stream := test.newStream("published")
		stream.Token.Set(stream.StreamID.Hex())
		stream.Fingerprint.Set(model.StreamFingerprintPending)
		require.NoError(t, test.session.Collection("Stream").Save(&stream, "test setup"))
	}

	pending := func() int64 {
		count, err := test.session.Collection("Stream").Count(exp.Equal("fingerprint", model.StreamFingerprintPending))
		require.NoError(t, err)
		return count
	}

	lastID, done, err := test.service.FillFingerprints(test.session, primitive.NilObjectID)
	require.NoError(t, err)
	require.False(t, done, "a full page may not be the last")
	require.Equal(t, int64(1), pending())

	_, done, err = test.service.FillFingerprints(test.session, lastID)
	require.NoError(t, err)
	require.True(t, done)
	require.Zero(t, pending())
}

// TestStream_FillFingerprints_SkipsAStreamItCannotBuild confirms that a Stream whose Template is
// missing keeps its placeholder, and does not stop the Streams after it from being filled
func TestStream_FillFingerprints_SkipsAStreamItCannotBuild(t *testing.T) {

	test := newOutboxSyncTest(t)

	broken := test.savePending(t, "published")
	require.NoError(t, queries.RawUpdate(t.Context(), test.session.Collection("Stream"), exp.Equal("_id", broken.StreamID), bson.M{
		"$set": bson.M{"templateId": "missing-template"},
	}))

	healthy := test.savePending(t, "published")

	_, done, err := test.service.FillFingerprints(test.session, primitive.NilObjectID)
	require.NoError(t, err)
	require.True(t, done)

	require.Equal(t, model.StreamFingerprintPending, test.fingerprint(t, broken.StreamID))
	require.Len(t, test.fingerprint(t, healthy.StreamID), 64)
}

// TestStream_ClaimFingerprint_KeepsARecordedFingerprint confirms that the fill never writes over a
// fingerprint that the outbox task recorded after the page was read
func TestStream_ClaimFingerprint_KeepsARecordedFingerprint(t *testing.T) {

	test := newOutboxSyncTest(t)

	stream := test.savePending(t, "published")

	// `stream` still holds the placeholder it was read with
	stale := storedStream(t, test.service, test.session, stream.StreamID)
	test.setFingerprint(t, stream.StreamID, "recorded-meanwhile")

	claimed, err := test.service.claimFingerprint(test.session, &stale, "from-a-stale-page")
	require.NoError(t, err)
	require.False(t, claimed)
	require.Equal(t, "recorded-meanwhile", test.fingerprint(t, stream.StreamID))
}

// TestStream_SyncOutbox_Placeholder confirms that the outbox task reads a placeholder as an older
// copy, so that an edit made before the fill reaches a Stream is still sent
func TestStream_SyncOutbox_Placeholder(t *testing.T) {

	t.Run("in the outbox", func(t *testing.T) {
		test := newOutboxSyncTest(t)
		stream := test.savePending(t, "published")

		loaded := storedStream(t, test.service, test.session, stream.StreamID)
		loaded.Label = "No Fate"
		require.NoError(t, test.service.Save(test.session, &loaded, "edited"))

		require.Equal(t, []string{vocab.ActivityTypeUpdate}, test.sync(t))
		require.Len(t, test.fingerprint(t, stream.StreamID), 64)
	})

	t.Run("leaving the outbox", func(t *testing.T) {
		test := newOutboxSyncTest(t)
		stream := test.savePending(t, "published")

		loaded := storedStream(t, test.service, test.session, stream.StreamID)
		loaded.StateID.Set("hidden")
		require.NoError(t, test.service.Save(test.session, &loaded, "withdrawn"))

		require.Equal(t, []string{vocab.ActivityTypeDelete}, test.sync(t))
		require.Empty(t, test.fingerprint(t, stream.StreamID))
	})

	t.Run("deleted", func(t *testing.T) {
		test := newOutboxSyncTest(t)
		stream := test.savePending(t, "published")

		loaded := storedStream(t, test.service, test.session, stream.StreamID)
		require.NoError(t, test.service.Delete(test.session, &loaded, "deleted"))

		require.Equal(t, []string{vocab.ActivityTypeDelete}, test.sync(t))
	})
}
