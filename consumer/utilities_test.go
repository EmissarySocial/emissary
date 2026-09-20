package consumer

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
)

// TestGetHostnameFromArgs documents the contract that every WithFactory-backed
// task enqueue must satisfy: WithFactory resolves the tenant Factory from the
// hostname returned here, and hard-fails the task when it is empty. A task
// enqueued with only a "url" argument (as the reply-tree crawlers once were)
// yields no hostname and can never run.
func TestGetHostnameFromArgs(t *testing.T) {

	// A "hostname" argument is used directly (reduced to its hostname).
	require.Equal(t, "example.com", getHostnameFromArgs(mapof.Any{"hostname": "https://example.com/@alice"}))
	require.Equal(t, "example.com", getHostnameFromArgs(mapof.Any{"hostname": "example.com"}))

	// LEGACY: the old "host" argument still resolves, so tasks queued before the
	// host->hostname migration continue to drain.
	require.Equal(t, "example.com", getHostnameFromArgs(mapof.Any{"host": "https://example.com/@alice"}))

	// An "actor" argument is used as a fallback source of the hostname.
	require.Equal(t, "example.com", getHostnameFromArgs(mapof.Any{"actor": "https://example.com/@alice"}))

	// "hostname" takes precedence over "actor".
	require.Equal(t, "host.example", getHostnameFromArgs(mapof.Any{
		"hostname": "https://host.example/@alice",
		"actor":    "https://actor.example/@bob",
	}))

	// A "url"-only map (the reply-tree crawler bug) yields NO hostname, which is
	// what caused those tasks to hard-fail with "Missing 'hostname' argument".
	require.Equal(t, "", getHostnameFromArgs(mapof.Any{"url": "http://localhost/6a4daccdc20bb1b4d44b8f94"}))

	// An empty map yields no hostname.
	require.Equal(t, "", getHostnameFromArgs(mapof.Any{}))
}

// TestRequeue pins the shared retry policy that PollFollowing_Record and every HTTP-backed
// consumer share: a 429 is rescheduled after the server's own Retry-After, any other 4xx is
// permanent, and everything else is retryable.
func TestRequeue(t *testing.T) {

	// No error is a success
	require.Equal(t, queue.ResultStatusSuccess, requeue(nil).Status)

	// RULE: 429 is retryable, and carries the delay the remote server asked for
	tooMany := tooManyRequests("90")
	result := requeue(tooMany)
	require.Equal(t, queue.ResultStatusRequeue, result.Status)
	require.Equal(t, 90*time.Second, result.Delay.Truncate(time.Second))

	// A 429 with no Retry-After still requeues, on derp's own default
	noHeader := requeue(tooManyRequests(""))
	require.Equal(t, queue.ResultStatusRequeue, noHeader.Status)
	require.Greater(t, noHeader.Delay, time.Duration(0))

	// RULE: Every other 4xx is permanent -- retrying cannot change the answer.  The BadRequest
	// here is exactly what hannibal.streams.document.Load returns for a relative ID (BUG-146).
	require.Equal(t, queue.ResultStatusFailure, requeue(derp.BadRequest("test", "Document ID is not a valid URL")).Status)
	require.Equal(t, queue.ResultStatusFailure, requeue(derp.NotFound("test", "Gone for good")).Status)
	require.Equal(t, queue.ResultStatusFailure, requeue(derp.Forbidden("test", "Nope")).Status)

	// 5xx and unclassified errors may succeed later
	require.Equal(t, queue.ResultStatusError, requeue(derp.Internal("test", "Server exploded")).Status)
	require.Equal(t, queue.ResultStatusError, requeue(errors.New("some transport failure")).Status)
}

// TestRequeue_WrappedTooManyRequests verifies that a 429 survives the derp.Wrap that every call
// site applies. PollFollowing_Record wraps before requeueing, so an unwrapped-only check would
// silently turn every rate limit into a permanent failure.
func TestRequeue_WrappedTooManyRequests(t *testing.T) {

	wrapped := derp.Wrap(tooManyRequests("30"), "test", "Loading document", "following: https://x.social/@bob")

	result := requeue(wrapped)
	require.Equal(t, queue.ResultStatusRequeue, result.Status)
	require.Equal(t, 30*time.Second, result.Delay.Truncate(time.Second))
}

// htmlInsteadOfActivityPub builds the error that benpate/remote returns when a remote server
// answers an ActivityPub request with 200 and an HTML body
func htmlInsteadOfActivityPub(url string) error {

	// This mirrors remote.Transaction.decodeResponseBody's ContentTypeHTML branch.  The
	// WithInternalError option is what makes the result a 500 even though the response was 200.
	response := &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
		Body:       io.NopCloser(strings.NewReader("<!DOCTYPE html><html><body>Hello</body></html>")),
	}

	request, _ := http.NewRequest(http.MethodGet, url, nil)

	return derp.Wrap(
		derp.NewHTTPError(request, response),
		"remote.Transaction.decodeResponseBody",
		"HTML must be read into an io.Writer, *string, or *byte[]",
		derp.WithInternalError(),
	)
}

// tooManyRequests builds the 429 shape that benpate/remote returns, optionally carrying a
// Retry-After header of the provided value
func tooManyRequests(retryAfter string) derp.HTTPError {

	header := http.Header{}

	if retryAfter != "" {
		header.Set("Retry-After", retryAfter)
	}

	return derp.HTTPError{
		Response: derp.HTTPResponseReport{
			StatusCode: http.StatusTooManyRequests,
			Header:     header,
		},
	}
}
