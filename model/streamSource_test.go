package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestStreamSourceSchema verifies that every StreamSource property round-trips through the schema
func TestStreamSourceSchema(t *testing.T) {

	streamSource := NewStreamSource()
	s := schema.New(StreamSourceSchema())

	table := []tableTestItem{
		{"streamSourceId", "123456781234567812345678", nil},
		{"streamId", "876543218765432187654321", nil},
		{"method", StreamSourceMethodHTTPS, nil},
		{"url", "https://example.com/org/repo", nil},
		{"config.branch", "main", nil},
		{"config.path", "docs/getting-started.md", nil},
		{"version", "0123456789abcdef0123456789abcdef01234567", nil},
		{"contentHash", "0123456789abcdef", nil},
		{"status", StreamSourceStatusSuccess, nil},
		{"statusMessage", "STATUS-MESSAGE", nil},
		{"lastSynced", "123", int64(123)},
	}

	tableTest_Schema(t, &s, &streamSource, table)
}

// TestNewStreamSource verifies the defaults that a new record starts with
func TestNewStreamSource(t *testing.T) {

	streamSource := NewStreamSource()

	require.False(t, streamSource.StreamSourceID.IsZero())
	require.Equal(t, streamSource.StreamSourceID.Hex(), streamSource.ID())
	require.Equal(t, StreamSourceStatusNew, streamSource.Status)
	require.NotNil(t, streamSource.Config, "Config must be writable without a nil-map panic")
	require.True(t, streamSource.StreamID.IsZero(), "a new record is not attached to any Stream")
}

// TestStreamSourceSchema_Rejects verifies that the schema refuses values a record must never hold
func TestStreamSourceSchema_Rejects(t *testing.T) {

	s := schema.New(StreamSourceSchema())

	// valid returns a record that passes validation, for each case to break in one place
	valid := func() StreamSource {
		result := NewStreamSource()
		result.StreamID = primitive.NewObjectID()
		result.Method = StreamSourceMethodHTTPS
		result.URL = "https://example.com/org/repo"
		return result
	}

	t.Run("baseline passes", func(t *testing.T) {
		streamSource := valid()
		_, err := s.Validate(&streamSource)
		require.Nil(t, err)
	})

	t.Run("unknown method", func(t *testing.T) {
		streamSource := valid()
		streamSource.Method = "FTP"
		_, err := s.Validate(&streamSource)
		require.NotNil(t, err)
	})

	t.Run("missing method", func(t *testing.T) {
		streamSource := valid()
		streamSource.Method = ""
		_, err := s.Validate(&streamSource)
		require.NotNil(t, err)
	})

	t.Run("missing url", func(t *testing.T) {
		streamSource := valid()
		streamSource.URL = ""
		_, err := s.Validate(&streamSource)
		require.NotNil(t, err)
	})

	t.Run("unknown status", func(t *testing.T) {
		streamSource := valid()
		streamSource.Status = "PAUSED"
		_, err := s.Validate(&streamSource)
		require.NotNil(t, err)
	})
}
