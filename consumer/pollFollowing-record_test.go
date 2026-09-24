package consumer

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
)

// Every branch of actorError but the rate-limited one writes to the Following record, which needs
// a live *service.Factory. What it records, and which status that becomes, is
// Following.SetStatusPollError's decision, and is tested in the service package.

// TestActorError_RateLimited confirms that a 429 requeues the task WITHOUT touching the Following.
// The factory is nil on purpose: reaching the service at all would panic.
func TestActorError_RateLimited(t *testing.T) {

	following := model.NewFollowing()

	// RULE: 429 is a CLIENT error too, so it must be caught before anything records a failure
	for _, err := range []error{tooManyRequests("45"), tooManyRequests(""), derp.Wrap(tooManyRequests("45"), "test", "wrapped")} {
		result := actorError(nil, nil, &following, err)
		require.Equal(t, queue.ResultStatusRequeue, result.Status)
	}

	require.Equal(t, model.FollowingStatusNew, following.Status, "a rate limit is the host's throttle, not this record's failure")
	require.Zero(t, following.ErrorCount)
}

// TestAnsweredWithoutActor pins the one signal that separates "the server answered, and the BODY
// was the problem" from every other failure: a 2xx status on the response that actually arrived.
func TestAnsweredWithoutActor(t *testing.T) {

	// Every 2xx counts, because any of them means the request completed
	for code := 200; code <= 299; code++ {
		require.True(t, answeredWithoutActor(answeredWith(code, "text/html")), "code %d", code)
	}

	// RULE: Nothing outside 2xx is this case. A 4xx or 5xx is a refusal or a server fault, and
	// a transport failure never produced a response at all.
	for _, code := range []int{0, 100, 301, 400, 404, 410, 429, 500, 503} {
		require.False(t, answeredWithoutActor(answeredWith(code, "text/html")), "code %d", code)
	}

	// An error carrying no HTTPError at all is never this case
	require.False(t, answeredWithoutActor(errors.New("dial tcp: lookup gone.example: no such host")))
	require.False(t, answeredWithoutActor(derp.Internal("test", "Server exploded")))

	// A nil error must not panic
	require.NotPanics(t, func() { answeredWithoutActor(nil) })
}

// TestAnsweredWithoutActor_IsWhatRemoteProduces drives benpate/remote against a live server that
// answers HTML, so the rule above is pinned to what the library ACTUALLY returns.
func TestAnsweredWithoutActor_IsWhatRemoteProduces(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body><h1>This domain is for sale</h1></body></html>`))
	}))

	defer server.Close()

	result := mapof.NewAny()
	transaction := remote.Get(server.URL).Accept("application/activity+json").Result(&result)
	transaction.AllowPrivateIPs(true)

	err := transaction.Send()
	require.NotNil(t, err, "a parked domain answering HTML must still be an error")
	require.True(t, answeredWithoutActor(err), "remote no longer carries the response status; the rule needs rewriting")
	require.False(t, shouldReportPollError(err))
}

// TestShouldReportPollError pins which failed polls reach the error log. A poll that fails for a
// reason this code already understands is recorded on the Following, and reporting it as well
// re-files the same fact on every cycle -- 1,891 records, 14.5% of the log, for five follows.
func TestShouldReportPollError(t *testing.T) {

	// RULE: A 2xx that carried no Actor is understood. The record says so; the log adds nothing.
	require.False(t, shouldReportPollError(answeredWith(http.StatusOK, "text/html")))
	require.False(t, shouldReportPollError(answeredWith(http.StatusNoContent, "")))

	// RULE: A 4xx is permanent and understood, and was already silent before this change
	require.False(t, shouldReportPollError(httpStatus(http.StatusNotFound)))
	require.False(t, shouldReportPollError(httpStatus(http.StatusUnauthorized)))
	require.False(t, shouldReportPollError(derp.Forbidden("test", "nope")))

	// RULE: Everything unexplained stays visible. A real 5xx, a DNS failure, a TLS failure and a
	// timeout are all things a human might need to act on.
	require.True(t, shouldReportPollError(httpStatus(http.StatusInternalServerError)))
	require.True(t, shouldReportPollError(httpStatus(http.StatusBadGateway)))
	require.True(t, shouldReportPollError(errors.New("dial tcp: lookup gone.example: no such host")))
	require.True(t, shouldReportPollError(errors.New("tls: handshake failure")))
	require.True(t, shouldReportPollError(derp.Internal("test", "Server exploded")))

	// A nil error is never passed here, but must not panic if it ever is
	require.NotPanics(t, func() { shouldReportPollError(nil) })
}

// httpStatus builds the derp.HTTPError shape that benpate/remote returns for one status code,
// which is exactly how a remote refusal reaches PollFollowing_Record in production
func httpStatus(code int) derp.HTTPError {
	return derp.HTTPError{
		Response: derp.HTTPResponseReport{StatusCode: code},
	}
}

// answeredWith builds the error that remote.Transaction.decodeResponseBody returns when a response
// arrives intact but cannot be decoded: an HTTPError carrying the real status, wrapped with
// WithInternalError, which is what makes the OUTER code 500 while the response underneath says 200.
func answeredWith(code int, contentType string) error {

	header := http.Header{}

	if contentType != "" {
		header.Set("Content-Type", contentType)
	}

	inner := derp.HTTPError{
		Response: derp.HTTPResponseReport{
			StatusCode: code,
			Status:     strconv.Itoa(code) + " " + http.StatusText(code),
			Header:     header,
		},
	}

	return derp.Wrap(inner, "remote.Transaction.decodeResponseBody", "HTML must be read into an io.Writer, *string, or *byte[]", derp.WithInternalError())
}
