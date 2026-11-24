package model

import (
	"encoding/json"
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// TestStream_OutboxFields_NewStream confirms that a new Stream is outside the outbox, and that followers hold nothing
func TestStream_OutboxFields_NewStream(t *testing.T) {
	stream := NewStream()
	require.False(t, stream.IsOutbox.Value())
	require.Empty(t, stream.Fingerprint)
}

// TestStream_OutboxFields_BSON confirms that both fields survive a round trip through BSON, under their stored names
func TestStream_OutboxFields_BSON(t *testing.T) {

	original := NewStream()
	original.IsOutbox.Set(true)
	original.Fingerprint.Set("abc123")

	encoded, err := bson.Marshal(original)
	require.NoError(t, err)

	decoded := NewStream()
	require.NoError(t, bson.Unmarshal(encoded, &decoded))
	require.True(t, decoded.IsOutbox.Value())
	require.Equal(t, "abc123", decoded.Fingerprint.Value())

	raw := bson.M{}
	require.NoError(t, bson.Unmarshal(encoded, &raw))
	require.Equal(t, true, raw["isOutbox"])
	require.Equal(t, "abc123", raw["fingerprint"])
}

// TestStream_OutboxFields_BSONDefaults confirms that the flag is always stored, so a query can match FALSE,
// and that an empty fingerprint is not stored at all
func TestStream_OutboxFields_BSONDefaults(t *testing.T) {

	encoded, err := bson.Marshal(NewStream())
	require.NoError(t, err)

	raw := bson.M{}
	require.NoError(t, bson.Unmarshal(encoded, &raw))
	require.Equal(t, false, raw["isOutbox"])
	require.NotContains(t, raw, "fingerprint")
}

// TestStream_OutboxFields_BSONMissing confirms that a record saved before the fields existed reads as never sent
func TestStream_OutboxFields_BSONMissing(t *testing.T) {

	encoded, err := bson.Marshal(bson.M{"stateId": "published"})
	require.NoError(t, err)

	decoded := Stream{}
	require.NoError(t, bson.Unmarshal(encoded, &decoded))
	require.False(t, decoded.IsOutbox.Value())
	require.Empty(t, decoded.Fingerprint)
}

// TestStream_OutboxFields_JSON confirms that the flag is written as "isOutbox", and that the fingerprint never leaves the server
func TestStream_OutboxFields_JSON(t *testing.T) {

	stream := NewStream()
	stream.IsOutbox.Set(true)
	stream.Fingerprint.Set("abc123")

	encoded, err := json.Marshal(stream)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"isOutbox":true`)
	require.NotContains(t, string(encoded), "abc123")
	require.NotContains(t, string(encoded), "fingerprint")

	// A fingerprint in incoming JSON is ignored
	decoded := NewStream()
	require.NoError(t, json.Unmarshal([]byte(`{"isOutbox":true, "fingerprint":"forged"}`), &decoded))
	require.True(t, decoded.IsOutbox.Value())
	require.Empty(t, decoded.Fingerprint)
}

// TestStream_Update_CopiesOutboxFields confirms that Update carries both fields, in both directions
func TestStream_Update_CopiesOutboxFields(t *testing.T) {

	source := NewStream()
	source.IsOutbox.Set(true)
	source.Fingerprint.Set("abc123")

	target := NewStream()
	target.Update(source)
	require.True(t, target.IsOutbox.Value())
	require.Equal(t, "abc123", target.Fingerprint.Value())

	// An Update from a Stream outside the outbox clears them
	target.Update(NewStream())
	require.False(t, target.IsOutbox.Value())
	require.Empty(t, target.Fingerprint)
}

// TestStream_OutboxFields_NoFormCanWrite confirms that the Stream's schema offers neither field
func TestStream_OutboxFields_NoFormCanWrite(t *testing.T) {

	streamSchema := schema.New(StreamSchema())

	for _, path := range []string{"outbox", "isOutbox", "fingerprint"} {
		t.Run(path, func(t *testing.T) {
			stream := NewStream()
			require.Error(t, streamSchema.Set(&stream, path, "true"))
			require.False(t, stream.IsOutbox.Value())
			require.Empty(t, stream.Fingerprint)

			_, err := streamSchema.Get(&stream, path)
			require.Error(t, err)
		})
	}
}

// TestStreamSummary_IsOutbox confirms that a summary loads the flag from the Stream's stored name
func TestStreamSummary_IsOutbox(t *testing.T) {

	require.Contains(t, StreamSummaryFields(), "isOutbox")
	require.NotContains(t, StreamSummaryFields(), "fingerprint")

	stream := NewStream()
	stream.IsOutbox.Set(true)

	encoded, err := bson.Marshal(stream)
	require.NoError(t, err)

	summary := NewStreamSummary()
	require.NoError(t, bson.Unmarshal(encoded, &summary))
	require.True(t, summary.IsOutbox)
}

// TestStream_MarkSaved confirms that MarkSaved makes the current State and flag the baseline for the next comparison
func TestStream_MarkSaved(t *testing.T) {

	stream := NewStream()
	stream.StateID.Set("published")
	stream.IsOutbox.Set(true)
	require.True(t, stream.StateID.IsChanged())
	require.True(t, stream.IsOutbox.IsChanged())

	stream.MarkSaved()
	require.Equal(t, "published", stream.StateID.Value())
	require.True(t, stream.IsOutbox.Value())
	require.False(t, stream.StateID.IsChanged())
	require.False(t, stream.IsOutbox.IsChanged())
	require.True(t, stream.IsOutbox.Original())
}

// TestStream_StateID_TracksItsLoadedValue confirms that a Stream read from BSON or JSON knows the State it was read in
func TestStream_StateID_TracksItsLoadedValue(t *testing.T) {

	original := NewStream()
	original.StateID.Set("published")
	original.IsOutbox.Set(true)

	encoded, err := bson.Marshal(original)
	require.NoError(t, err)

	loaded := NewStream()
	require.NoError(t, bson.Unmarshal(encoded, &loaded))
	require.False(t, loaded.StateID.IsChanged())
	require.False(t, loaded.IsOutbox.IsChanged())

	loaded.SetState("default")
	require.True(t, loaded.StateID.IsChanged())
	require.Equal(t, "published", loaded.StateID.Original())
	require.Equal(t, "default", loaded.State())
}
