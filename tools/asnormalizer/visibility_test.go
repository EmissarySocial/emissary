package asnormalizer

import (
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/stretchr/testify/require"
)

const (
	testPublic    = "https://www.w3.org/ns/activitystreams#Public" // the Public collection's address
	testFollowers = "https://example.com/users/ben/followers"      // the test author's followers collection
)

// noteTo returns a Note addressed to the given "to" and "cc" values, either of which may be nil.
func noteTo(to any, cc any) streams.Document {

	value := map[string]any{"type": "Note", "id": "https://example.com/n/1", "content": "hi"}

	if to != nil {
		value["to"] = to
	}

	if cc != nil {
		value["cc"] = cc
	}

	return streams.NewDocument(value)
}

// TestVisibility works through each way a post can be addressed, in the shapes servers actually send.
func TestVisibility(t *testing.T) {

	cases := []struct {
		name     string
		to       any
		cc       any
		expected string
	}{
		{"public, copied to followers", []any{testPublic}, []any{testFollowers}, "public"},
		{"public as a single value", testPublic, nil, "public"},
		{"public in the short spelling", []any{"as:Public"}, nil, "public"},
		{"unlisted", []any{testFollowers}, []any{testPublic}, "unlisted"},
		{"followers only", []any{testFollowers}, []any{"https://other.example/users/amy"}, "private"},
		{"direct to one person", []any{"https://other.example/users/amy"}, nil, "direct"},
		{"direct to two people", []any{"https://other.example/users/amy"}, []any{"https://third.example/users/bo"}, "direct"},
		{"addressed to nobody", nil, nil, ""},
	}

	for _, testCase := range cases {
		require.Equal(t, testCase.expected, Visibility(noteTo(testCase.to, testCase.cc), testFollowers), testCase.name)
	}

	// The author's followers address is recognised even when it does not end in "/followers"
	custom := noteTo([]any{"https://example.com/collections/fans"}, nil)
	require.Equal(t, "direct", Visibility(custom, ""), "unknown address with no hint")
	require.Equal(t, "private", Visibility(custom, "https://example.com/collections/fans"))
}

// TestObject_StoresTheAudienceAsOneWord confirms ingest keeps the result and not the recipient lists.
func TestObject_StoresTheAudienceAsOneWord(t *testing.T) {

	result := Object(nil, noteTo([]any{testFollowers}, []any{"https://other.example/users/amy"}))

	require.Equal(t, "private", result["visibility"])
	require.NotContains(t, result, "to")
	require.NotContains(t, result, "cc")

	require.NotContains(t, Object(nil, noteTo(nil, nil)), "visibility", "a post naming nobody stores nothing")
}
