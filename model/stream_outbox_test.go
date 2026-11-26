package model

import (
	"encoding/json"
	"testing"

	"github.com/benpate/delta"
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

// TestStream_MarkSaved_ResetsSyndication confirms that a save makes the saved targets the
// baseline, so that a second save of the same copy does not report the same changes again
func TestStream_MarkSaved_ResetsSyndication(t *testing.T) {

	stream := NewStream()
	stream.Syndication = delta.NewSlice("mastodon", "bluesky")
	stream.Syndication.Set([]string{"bluesky", "threads"})
	require.Equal(t, []string{"threads"}, stream.Syndication.Added())
	require.Equal(t, []string{"mastodon"}, stream.Syndication.Deleted())

	stream.MarkSaved()

	require.False(t, stream.Syndication.IsChanged())
	require.Empty(t, stream.Syndication.Added())
	require.Empty(t, stream.Syndication.Deleted())
	require.Equal(t, []string{"bluesky", "threads"}, stream.Syndication.Value())
}

// TestStream_Syndication_LegacyBSON confirms that a Stream stored by the old rosetta type,
// as {values: [...]}, loads unchanged and is saved back as a plain array
func TestStream_Syndication_LegacyBSON(t *testing.T) {

	stored, err := bson.Marshal(bson.M{"syndication": bson.M{"values": bson.A{"mastodon", "bluesky"}}})
	require.NoError(t, err)

	stream := NewStream()
	require.NoError(t, bson.Unmarshal(stored, &stream))
	require.Equal(t, []string{"mastodon", "bluesky"}, stream.Syndication.Value())
	require.False(t, stream.Syndication.IsChanged())

	saved, err := bson.Marshal(stream)
	require.NoError(t, err)
	require.Equal(t, bson.A{"mastodon", "bluesky"}, readBSONField(t, saved, "syndication"))

	// An empty list is left out of the document
	stream.Syndication.Clear()
	saved, err = bson.Marshal(stream)
	require.NoError(t, err)
	require.Nil(t, readBSONField(t, saved, "syndication"))
}

// readBSONField decodes a BSON document into a map, and returns one of its properties
func readBSONField(t *testing.T, document []byte, name string) any {
	t.Helper()

	result := bson.M{}
	require.NoError(t, bson.Unmarshal(document, &result))

	return result[name]
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

// TestStream_Toot_Date confirms that a status shows the publish date, and the create date for
// a Stream that has none
func TestStream_Toot_Date(t *testing.T) {

	const created = int64(1_700_000_000_000)

	for name, test := range map[string]struct {
		publishDate int64
		expected    string
	}{
		"published":     {publishDate: 1_750_000_000, expected: "2025-06-15T15:06:40.000Z"},
		"no date":       {publishDate: 0, expected: "2023-11-14T22:13:20.000Z"},
		"negative date": {publishDate: -1, expected: "2023-11-14T22:13:20.000Z"},
	} {
		t.Run(name, func(t *testing.T) {

			stream := NewStream()
			stream.CreateDate = created
			stream.PublishDate = test.publishDate

			require.Equal(t, test.expected, stream.Toot("https://example.com").CreatedAt)
		})
	}
}

// TestStream_IsPublished confirms that "published" means in the outbox and not deleted, whatever the dates say
func TestStream_IsPublished(t *testing.T) {

	for name, test := range map[string]struct {
		isOutbox    bool
		publishDate int64
		deleteDate  int64
		expected    bool
	}{
		"in the outbox":                  {isOutbox: true, publishDate: 100, expected: true},
		"in the outbox, with no date":    {isOutbox: true, publishDate: 0, expected: true},
		"on the site, not in the outbox": {isOutbox: false, publishDate: 100, expected: false},
		"a new Stream":                   {isOutbox: false, publishDate: 0, expected: false},
		"deleted":                        {isOutbox: true, publishDate: 100, deleteDate: 200, expected: false},
	} {
		t.Run(name, func(t *testing.T) {

			stream := NewStream()
			stream.IsOutbox.Set(test.isOutbox)
			stream.PublishDate = test.publishDate
			stream.DeleteDate = test.deleteDate

			require.Equal(t, test.expected, stream.IsPublished())

			// The virtual field and the syndication payload give the same answer
			value, ok := stream.GetBoolOK("isPublished")
			require.True(t, ok)
			require.Equal(t, test.expected, value)
			require.Equal(t, test.expected, stream.SyndicationData()["isPublished"])
		})
	}
}

// TestStreamSummary_IsPublished confirms that a summary is published when it is in the outbox, whatever the dates say
func TestStreamSummary_IsPublished(t *testing.T) {

	summary := StreamSummary{PublishDate: 100}
	require.False(t, summary.IsPublished(), "a date is not publication")

	summary = StreamSummary{IsOutbox: true}
	require.True(t, summary.IsPublished(), "the flag alone is publication")
}

// TestStream_NoUnpublishDate confirms that the schema and the stored record no longer hold "unpublishDate"
func TestStream_NoUnpublishDate(t *testing.T) {

	stream := NewStream()

	require.Error(t, schema.New(StreamSchema()).Set(&stream, "unpublishDate", 123))
	require.NotContains(t, StreamSummaryFields(), "unpublishDate")

	encoded, err := bson.Marshal(stream)
	require.NoError(t, err)

	raw := bson.M{}
	require.NoError(t, bson.Unmarshal(encoded, &raw))
	require.NotContains(t, raw, "unpublishDate")
	require.Equal(t, int64(0), raw["publishDate"])
}
