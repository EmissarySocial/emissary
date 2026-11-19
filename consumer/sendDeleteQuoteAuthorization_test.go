package consumer

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
)

// TestQuoteAuthor pins that the quoting actor is read from the quote post, whether the post names
// its author by ID or embeds the whole actor
func TestQuoteAuthor(t *testing.T) {

	client := mockClient{documents: mapof.Any{
		"https://remote.example/notes/1": mapof.Any{
			vocab.PropertyID:           "https://remote.example/notes/1",
			vocab.PropertyAttributedTo: "https://remote.example/users/bob",
		},
		"https://remote.example/notes/2": mapof.Any{
			vocab.PropertyID:           "https://remote.example/notes/2",
			vocab.PropertyAttributedTo: mapof.Any{vocab.PropertyID: "https://remote.example/users/carol"},
		},
		"https://remote.example/notes/3": mapof.Any{
			vocab.PropertyID: "https://remote.example/notes/3",
		},
	}}

	quoter, err := quoteAuthor(client, "https://remote.example/notes/1")
	require.NoError(t, err)
	require.Equal(t, "https://remote.example/users/bob", quoter)

	quoter, err = quoteAuthor(client, "https://remote.example/notes/2")
	require.NoError(t, err)
	require.Equal(t, "https://remote.example/users/carol", quoter, "an embedded actor")

	quoter, err = quoteAuthor(client, "https://remote.example/notes/3")
	require.NoError(t, err)
	require.Equal(t, "", quoter, "a post that names no author")

	quoter, err = quoteAuthor(client, "https://remote.example/notes/missing")
	require.Error(t, err)
	require.Equal(t, "", quoter)
}

// TestRetryQuoteLookup pins which failures to load the quote post retry the task, and which let the
// Delete go to the followers alone
func TestRetryQuoteLookup(t *testing.T) {

	// A post that loaded goes straight on
	_, retry := retryQuoteLookup(nil)
	require.False(t, retry)

	// A post that is gone, or closed to us, will never name its author
	for _, err := range []error{
		derp.NotFound("test", "gone"),
		derp.Forbidden("test", "closed"),
		derp.Unauthorized("test", "closed"),
	} {
		_, retry := retryQuoteLookup(err)
		require.False(t, retry, derp.ErrorCode(err))
	}

	// A server that is down may come back, so the task is retried and nothing is sent yet
	result, retry := retryQuoteLookup(derp.Internal("test", "connection refused"))
	require.True(t, retry)
	require.Equal(t, queue.ResultStatusError, result.Status)
	require.Error(t, result.Error)

	// A throttled server is a client error too, and must still be asked again
	result, retry = retryQuoteLookup(tooManyRequests(""))
	require.True(t, retry)
	require.Equal(t, queue.ResultStatusRequeue, result.Status)
	require.NoError(t, result.Error)
}
