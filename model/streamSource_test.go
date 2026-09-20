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
	require.Equal(t, StreamSourceMethodHTTPS, streamSource.Method, "the schema requires a Method and no form offers one")
}

// TestNewStreamSource_PassesValidationFromTheSettingsScreen builds a record the way the
// `with-stream-source` step does -- the constructor, the Stream it attaches to, and a URL typed
// into the form -- and confirms the schema accepts it.  Nothing else offers a value for `method`,
// so a constructor that left it empty made the Add-a-Source form impossible to submit.
func TestNewStreamSource_PassesValidationFromTheSettingsScreen(t *testing.T) {

	streamSource := NewStreamSource()
	streamSource.StreamID = primitive.NewObjectID()
	streamSource.URL = "https://raw.githubusercontent.com/example/repo/refs/heads/main/README.md"

	s := schema.New(StreamSourceSchema())
	_, err := s.Validate(&streamSource)
	require.Nil(t, err)
}

// TestStreamSourceSchema_Rejects verifies that the schema refuses values a record must never hold
func TestStreamSourceSchema_Rejects(t *testing.T) {

	s := schema.New(StreamSourceSchema())

	// valid returns a record that passes validation, for each case to break in one place
	// Only the two values a caller genuinely supplies are set here.  Setting Method as well
	// would re-hide the defect this helper once masked: the constructor left it EMPTY, so no
	// record created through the settings screen could pass validation.
	valid := func() StreamSource {
		result := NewStreamSource()
		result.StreamID = primitive.NewObjectID()
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

// TestIsValidWebhookToken confirms that only a bare token is accepted, never a whole URL
func TestIsValidWebhookToken(t *testing.T) {

	require.True(t, IsValidWebhookToken(NewWebhookToken()), "a generated token must always be valid")

	valid := []string{
		"abcdefghijklmnop",       // exactly the minimum length
		"my-shared-docs-token",   // dashes
		"my_shared_docs_token",   // underscores
		"docs.emissary.token.v2", // dots
		"MixedCase0123456789",    // letters and digits
	}

	for _, token := range valid {
		require.True(t, IsValidWebhookToken(token), token)
	}

	// RULE: the server and path are fixed, so anything shaped like an address is a pasted URL.
	invalid := []string{
		"",         // nothing at all
		"tooshort", // under the minimum length
		"https://example.com/.streamsource/webhook/abcdefgh", // the whole URL
		"example.com/.streamsource/webhook/abcdefghijklmnop", // a host and path
		"/.streamsource/webhook/abcdefghijklmnop",            // just the path
		"abcdefghijklmnop/extra",                             // a stray slash
		"abcdefghij klmnop",                                  // whitespace
		"abcdefghijklmnop?query=1",                           // a query string
		"abcdefghijklmnop#fragment",                          // a fragment
	}

	for _, token := range invalid {
		require.False(t, IsValidWebhookToken(token), token)
	}
}
