package ascache

import (
	"net/http"
	"testing"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// failingClient is an origin that answers every Load with the provided error code, counting calls
type failingClient struct {
	countingClient
	code int
}

// Load records the call and fails, or succeeds once code is cleared
func (client *failingClient) Load(uri string, options ...any) (streams.Document, error) {

	if client.code == 0 {
		return client.countingClient.Load(uri, options...)
	}

	client.calls++
	return streams.NilDocument(), derp.Internal("test.failingClient", "origin failed", derp.WithCode(client.code))
}

// newFailingTestClient returns a cache Client with a FailureMemo, in front of a failing origin
func newFailingTestClient(code int) (*Client, *failingClient, *FailureMemo) {

	origin := &failingClient{countingClient: countingClient{maxAge: 3600}, code: code}
	memo := NewFailureMemo(time.Minute)
	client := New(origin, nil, newFakeServer(), "Application", primitive.NilObjectID, "test.example", WithFailureMemo(memo))

	return client, origin, memo
}

// TestFailureMemo_Window confirms a failure is remembered for the window, then forgotten
func TestFailureMemo_Window(t *testing.T) {

	now := time.Now()
	memo := NewFailureMemo(time.Minute)
	memo.now = func() time.Time { return now }

	memo.remember("https://down.example/1")
	require.True(t, memo.failedRecently("https://down.example/1"))
	require.False(t, memo.failedRecently("https://down.example/2"))

	now = now.Add(2 * time.Minute)
	require.False(t, memo.failedRecently("https://down.example/1"))
}

// TestFailureMemo_Limit confirms a full memo drops expired entries before it grows
func TestFailureMemo_Limit(t *testing.T) {

	now := time.Now()
	memo := NewFailureMemo(time.Minute)
	memo.now = func() time.Time { return now }

	for index := range failureMemoLimit {
		memo.until[string(rune(index))] = now.Add(-time.Second)
	}

	memo.remember("https://down.example/1")
	require.Len(t, memo.until, 1)
}

// TestClient_Load_ServerFailureIsRemembered confirms a 502 origin is contacted once per window
func TestClient_Load_ServerFailureIsRemembered(t *testing.T) {

	client, origin, _ := newFailingTestClient(http.StatusBadGateway)

	for range 5 {
		_, err := client.Load("https://down.example/post")
		require.Error(t, err)
		require.Equal(t, http.StatusBadGateway, derp.ErrorCode(err))
	}

	require.Equal(t, 1, origin.calls)
}

// TestClient_Load_ForbiddenIsRemembered confirms a server that refuses us is contacted once per window
func TestClient_Load_ForbiddenIsRemembered(t *testing.T) {

	client, origin, _ := newFailingTestClient(http.StatusForbidden)

	for range 5 {
		_, err := client.Load("https://blocking.example/@alice")
		require.Error(t, err)
	}

	require.Equal(t, 1, origin.calls)
}

// TestClient_Load_NotFoundIsNotRemembered confirms a 4xx answer is never treated as an outage
func TestClient_Load_NotFoundIsNotRemembered(t *testing.T) {

	client, origin, memo := newFailingTestClient(http.StatusNotFound)

	for range 3 {
		_, err := client.Load("https://up.example/missing")
		require.Error(t, err)
	}

	require.Equal(t, 3, origin.calls)
	require.False(t, memo.failedRecently("https://up.example/missing"))
}

// TestClient_Load_ForcedReloadRetries confirms a forced reload still reaches a failed origin, and a success clears it
func TestClient_Load_ForcedReloadRetries(t *testing.T) {

	client, origin, memo := newFailingTestClient(http.StatusBadGateway)

	_, err := client.Load("https://down.example/post")
	require.Error(t, err)

	origin.code = 0
	_, err = client.Load("https://down.example/post", WithWriteOnly(), WithMinAge(0))
	require.NoError(t, err)

	require.Equal(t, 2, origin.calls)
	require.False(t, memo.failedRecently("https://down.example/post"))
}
