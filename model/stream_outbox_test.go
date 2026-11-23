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
	require.False(t, stream.IsOutbox)
	require.Empty(t, stream.OutboxFingerprint)
}

// TestStream_OutboxFields_BSON confirms that both fields survive a round trip through BSON, under their stored names
func TestStream_OutboxFields_BSON(t *testing.T) {

	original := NewStream()
	original.IsOutbox = true
	original.OutboxFingerprint = "abc123"

	encoded, err := bson.Marshal(original)
	require.NoError(t, err)

	decoded := NewStream()
	require.NoError(t, bson.Unmarshal(encoded, &decoded))
	require.True(t, decoded.IsOutbox)
	require.Equal(t, "abc123", decoded.OutboxFingerprint)

	raw := bson.M{}
	require.NoError(t, bson.Unmarshal(encoded, &raw))
	require.Equal(t, true, raw["isOutbox"])
	require.Equal(t, "abc123", raw["outboxFingerprint"])
}

// TestStream_OutboxFields_BSONDefaults confirms that the flag is always stored, so a query can match FALSE,
// and that an empty fingerprint is not stored at all
func TestStream_OutboxFields_BSONDefaults(t *testing.T) {

	encoded, err := bson.Marshal(NewStream())
	require.NoError(t, err)

	raw := bson.M{}
	require.NoError(t, bson.Unmarshal(encoded, &raw))
	require.Equal(t, false, raw["isOutbox"])
	require.NotContains(t, raw, "outboxFingerprint")
}

// TestStream_OutboxFields_BSONMissing confirms that a record saved before the fields existed reads as never sent
func TestStream_OutboxFields_BSONMissing(t *testing.T) {

	encoded, err := bson.Marshal(bson.M{"stateId": "published"})
	require.NoError(t, err)

	decoded := Stream{}
	require.NoError(t, bson.Unmarshal(encoded, &decoded))
	require.False(t, decoded.IsOutbox)
	require.Empty(t, decoded.OutboxFingerprint)
}

// TestStream_OutboxFields_JSON confirms that the flag is written as "isOutbox", and that the fingerprint never leaves the server
func TestStream_OutboxFields_JSON(t *testing.T) {

	stream := NewStream()
	stream.IsOutbox = true
	stream.OutboxFingerprint = "abc123"

	encoded, err := json.Marshal(stream)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"isOutbox":true`)
	require.NotContains(t, string(encoded), "abc123")
	require.NotContains(t, string(encoded), "outboxFingerprint")

	// A fingerprint in incoming JSON is ignored
	decoded := NewStream()
	require.NoError(t, json.Unmarshal([]byte(`{"isOutbox":true, "outboxFingerprint":"forged"}`), &decoded))
	require.True(t, decoded.IsOutbox)
	require.Empty(t, decoded.OutboxFingerprint)
}

// TestStream_Update_CopiesOutboxFields confirms that Update carries both fields, in both directions
func TestStream_Update_CopiesOutboxFields(t *testing.T) {

	source := NewStream()
	source.IsOutbox = true
	source.OutboxFingerprint = "abc123"

	target := NewStream()
	target.Update(source)
	require.True(t, target.IsOutbox)
	require.Equal(t, "abc123", target.OutboxFingerprint)

	// An Update from a Stream outside the outbox clears them
	target.Update(NewStream())
	require.False(t, target.IsOutbox)
	require.Empty(t, target.OutboxFingerprint)
}

// TestStream_OutboxFields_NoFormCanWrite confirms that the Stream's schema offers neither field
func TestStream_OutboxFields_NoFormCanWrite(t *testing.T) {

	streamSchema := schema.New(StreamSchema())

	for _, path := range []string{"outbox", "isOutbox", "outboxFingerprint"} {
		t.Run(path, func(t *testing.T) {
			stream := NewStream()
			require.Error(t, streamSchema.Set(&stream, path, "true"))
			require.False(t, stream.IsOutbox)
			require.Empty(t, stream.OutboxFingerprint)

			_, err := streamSchema.Get(&stream, path)
			require.Error(t, err)
		})
	}
}

// TestStreamSummary_IsOutbox confirms that a summary loads the flag from the Stream's stored name
func TestStreamSummary_IsOutbox(t *testing.T) {

	require.Contains(t, StreamSummaryFields(), "isOutbox")
	require.NotContains(t, StreamSummaryFields(), "outboxFingerprint")

	stream := NewStream()
	stream.IsOutbox = true

	encoded, err := bson.Marshal(stream)
	require.NoError(t, err)

	summary := NewStreamSummary()
	require.NoError(t, bson.Unmarshal(encoded, &summary))
	require.True(t, summary.IsOutbox)
}
