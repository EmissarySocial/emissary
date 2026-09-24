package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

/******************************************
 * SetStatusPollError
 ******************************************/

// TestSetStatusPollError_Gone confirms that only the remote server's own 410 marks a Following GONE
func TestSetStatusPollError_Gone(t *testing.T) {

	for _, err := range []error{httpStatus(http.StatusGone), derp.Gone("test", "deleted")} {

		service, session, collection := newPollErrorService()
		following := model.NewFollowing()

		require.NoError(t, service.SetStatusPollError(session, &following, err))
		require.Equal(t, model.FollowingStatusGone, following.Status)
		require.Equal(t, "This account has been deleted (410).", following.StatusMessage)
		require.Len(t, collection.saved, 1)
	}
}

// TestSetStatusPollError_Failure confirms that every other failure is RECORDED, including the
// dead-domain shapes that answer no HTTP status at all
func TestSetStatusPollError_Failure(t *testing.T) {

	cases := []error{

		// RULE: 404 is NOT 410. Misconfiguration produces a missing document often, and BUG-148's
		// own Defect B was a 404 on a key that this server was wrongly withholding.
		httpStatus(http.StatusNotFound),

		// RULE: 401/403 never shortcut to dead. Defect B's 401s were OUR defect, and abandoning
		// those follows would have made them unrecoverable once the key was fixed.
		httpStatus(http.StatusUnauthorized),
		httpStatus(http.StatusForbidden),

		// RULE: The dead-domain shapes must be RECORDED, not retried
		errors.New("dial tcp: lookup gone.example: no such host"),
		errors.New("tls: handshake failure"),
		errors.New("context deadline exceeded"),
		derp.Internal("test", "Server exploded"),

		// A server that answered with something other than an Actor
		answeredWith(http.StatusOK, "text/html"),
	}

	for _, err := range cases {

		service, session, collection := newPollErrorService()
		following := model.NewFollowing()

		require.NoError(t, service.SetStatusPollError(session, &following, err))
		require.Equal(t, model.FollowingStatusFailure, following.Status, "%v", err)
		require.Equal(t, followingStatusMessage(err), following.StatusMessage)
		require.Equal(t, 1, following.ErrorCount)
		require.Len(t, collection.saved, 1)
	}
}

// TestSetStatusPollError_EveryCode confirms that 410 is the only status with special treatment.
func TestSetStatusPollError_EveryCode(t *testing.T) {

	// A 429 is recorded too if it ever arrives here, which is why callers requeue it first

	for code := range 600 {

		service, session, _ := newPollErrorService()
		following := model.NewFollowing()

		require.NoError(t, service.SetStatusPollError(session, &following, httpStatus(code)))

		if code == http.StatusGone {
			require.Equal(t, model.FollowingStatusGone, following.Status, "code %d", code)
			continue
		}

		require.Equal(t, model.FollowingStatusFailure, following.Status, "code %d", code)
	}
}

// TestSetStatusPollError_Pauses confirms that a failure recorded here still escalates to PAUSED
// once the source has been failing long enough, and often enough
func TestSetStatusPollError_Pauses(t *testing.T) {

	service, session, _ := newPollErrorService()
	following := model.NewFollowing()
	following.ErrorCount = unresponsiveAfterErrors - 1
	following.LastPolled = time.Now().Unix() - unresponsiveAfterSeconds - 60

	require.NoError(t, service.SetStatusPollError(session, &following, errors.New("no such host")))
	require.Equal(t, model.FollowingStatusPaused, following.Status)
}

// TestSetStatusPollError_BoundsTheMessage confirms the message fits model.Following's
// "statusMessage" schema, which rejects anything over 1024 bytes.
func TestSetStatusPollError_BoundsTheMessage(t *testing.T) {

	// A quoted root message has no bound of its own

	service, session, _ := newPollErrorService()
	following := model.NewFollowing()

	require.NoError(t, service.SetStatusPollError(session, &following, errors.New(strings.Repeat("日", 100_000))))
	require.LessOrEqual(t, len(following.StatusMessage), 1024)
	require.True(t, utf8.ValidString(following.StatusMessage))
	require.Contains(t, following.StatusMessage, "Could not reach this server")
}

// TestFollowing_StatusSetters_BoundTheMessage confirms that every setter taking a caller's message
// bounds it to the schema itself.
func TestFollowing_StatusSetters_BoundTheMessage(t *testing.T) {

	huge := strings.Repeat("x", 100_000)

	setters := map[string]func(Following, backoffSession, *model.Following) error{
		"SetStatusGone": func(s Following, session backoffSession, f *model.Following) error {
			return s.SetStatusGone(session, f, huge)
		},
		"SetStatusPollFailure": func(s Following, session backoffSession, f *model.Following) error {
			return s.SetStatusPollFailure(session, f, huge)
		},
		"SetStatusFailure": func(s Following, session backoffSession, f *model.Following) error {
			return s.SetStatusFailure(session, f, huge)
		},
	}

	for name, setter := range setters {

		service, session, _ := newPollErrorService()
		following := model.NewFollowing()

		require.NoError(t, setter(service, session, &following), name)
		require.Len(t, following.StatusMessage, 1024, name)
	}
}

/******************************************
 * followingStatusMessage
 ******************************************/

// TestFollowingStatusMessage checks the sentence that reaches the Following's owner, which names
// the HTTP code for anyone reporting it.
func TestFollowingStatusMessage(t *testing.T) {

	// A refusal reads as a refusal, and names its code
	require.Contains(t, followingStatusMessage(derp.Unauthorized("test", "nope")), "401")
	require.Contains(t, followingStatusMessage(derp.Unauthorized("test", "nope")), "refused")
	require.Contains(t, followingStatusMessage(derp.Forbidden("test", "nope")), "403")
	require.Contains(t, followingStatusMessage(derp.Forbidden("test", "nope")), "refused")

	// A missing actor reads as missing
	require.Contains(t, followingStatusMessage(derp.NotFound("test", "gone")), "404")
	require.Contains(t, followingStatusMessage(derp.NotFound("test", "gone")), "could not be found")

	// Every other client error falls through to the generic sentence, still naming the code
	require.Contains(t, followingStatusMessage(derp.BadRequest("test", "malformed")), "400")
	require.Contains(t, followingStatusMessage(derp.BadRequest("test", "malformed")), "Unable to read")

	// RULE: The message is never empty, for ANY error -- an empty StatusMessage renders as a
	// red "failed" badge with no explanation at all.
	for code := 0; code <= 599; code++ {
		message := followingStatusMessage(httpStatus(code))
		require.NotEmpty(t, message, "code %d", code)
	}

	// A nil error is never passed here, but must not panic if it ever is
	require.NotPanics(t, func() { followingStatusMessage(nil) })
	require.NotEmpty(t, followingStatusMessage(nil))

	// The wrapping that every call site applies must not hide the code
	wrapped := derp.Wrap(derp.Unauthorized("test", "nope"), "test", "Loading ActivityPub Actor")
	require.Contains(t, followingStatusMessage(wrapped), "401")
}

// TestFollowingStatusMessage_Transport covers the failures that never reach a server at all.
func TestFollowingStatusMessage_Transport(t *testing.T) {

	// RULE: derp reports a DNS or TLS failure as 500, like a real server error, so naming the code
	// would tell the owner "(500)" about a domain that does not resolve.

	message := followingStatusMessage(errors.New("dial tcp: lookup gone.example: no such host"))
	require.Contains(t, message, "Could not reach this server")
	require.Contains(t, message, "no such host")
	require.NotContains(t, message, "500")

	// A deleted account says so plainly, rather than "may have been moved or deleted"
	require.Equal(t, "This account has been deleted (410).", followingStatusMessage(httpStatus(http.StatusGone)))
}

// TestFollowingStatusMessage_AnsweredWithoutActor covers the sentence for a server that answered
// with something that is not an account.
func TestFollowingStatusMessage_AnsweredWithoutActor(t *testing.T) {

	message := followingStatusMessage(answeredWith(http.StatusOK, "text/html; charset=utf-8"))

	// RULE: Never claim the server was unreachable. It answered.
	require.NotContains(t, message, "Could not reach this server")
	require.NotContains(t, message, "200 OK", "quoting the status line as a reason is the defect")

	// It names what arrived, so the owner can tell a web page from a broken feed
	require.Contains(t, message, "200")
	require.Contains(t, message, "text/html")
	require.Contains(t, message, "did not return an account")

	// A feed body reaches the same sentence: while RSS is unsupported, it is equally "not an
	// account", and naming the media type is what distinguishes the two for anyone reporting it.
	feedMessage := followingStatusMessage(answeredWith(http.StatusOK, "application/rss+xml"))
	require.Contains(t, feedMessage, "application/rss+xml")
	require.NotContains(t, feedMessage, "Could not reach this server")

	// An unknown media type still produces a complete sentence
	bareMessage := followingStatusMessage(answeredWith(http.StatusOK, ""))
	require.NotEmpty(t, bareMessage)
	require.Contains(t, bareMessage, "200")

	// A genuine transport failure is untouched by this branch
	require.Contains(t, followingStatusMessage(errors.New("no such host")), "Could not reach this server")
}

/******************************************
 * answeredWithoutActor
 ******************************************/

// TestAnsweredWithoutActor pins the one signal that separates "the server answered, and the BODY
// was the problem" from every other failure: a 2xx status on the response that actually arrived.
func TestAnsweredWithoutActor(t *testing.T) {

	// A 2xx answer that could not be read as an Actor, with the media type it arrived as
	status, contentType, answered := answeredWithoutActor(answeredWith(http.StatusOK, "text/html; charset=utf-8"))
	require.True(t, answered)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "text/html", contentType, "the charset parameter must not reach the owner's sentence")

	// Every 2xx counts, because any of them means the request completed
	for code := 200; code <= 299; code++ {
		_, _, answered := answeredWithoutActor(answeredWith(code, "text/html"))
		require.True(t, answered, "code %d", code)
	}

	// RULE: Nothing outside 2xx is this case. A 4xx or 5xx is a refusal or a server fault, and
	// a transport failure never produced a response at all.
	for _, code := range []int{0, 100, 301, 400, 404, 410, 429, 500, 503} {
		_, _, answered := answeredWithoutActor(answeredWith(code, "text/html"))
		require.False(t, answered, "code %d", code)
	}

	// An error carrying no HTTPError at all is never this case
	_, _, answered = answeredWithoutActor(errors.New("dial tcp: lookup gone.example: no such host"))
	require.False(t, answered)

	_, _, answered = answeredWithoutActor(derp.Internal("test", "Server exploded"))
	require.False(t, answered)

	// A missing Content-Type is reported as empty rather than guessed at
	_, contentType, answered = answeredWithoutActor(answeredWith(http.StatusOK, ""))
	require.True(t, answered)
	require.Empty(t, contentType)

	// RULE: The media type is written by the REMOTE server, so it is bounded. Unbounded, it
	// would push the explanation out of the owner's status message.
	_, contentType, answered = answeredWithoutActor(answeredWith(http.StatusOK, strings.Repeat("x", 100_000)))
	require.True(t, answered)
	require.LessOrEqual(t, len(contentType), 100)

	// RULE: The bound never splits a multi-byte character, whatever byte it lands on
	for padding := 95; padding <= 100; padding++ {
		_, contentType, _ = answeredWithoutActor(answeredWith(http.StatusOK, strings.Repeat("a", padding)+"🦖🦖"))
		require.True(t, utf8.ValidString(contentType), "padding %d", padding)
		require.LessOrEqual(t, len(contentType), 100, "padding %d", padding)
	}

	// A nil error must not panic
	require.NotPanics(t, func() { answeredWithoutActor(nil) })
}

// TestAnsweredWithoutActor_IsWhatRemoteProduces drives benpate/remote against a live server that
// answers HTML, so the rule above is pinned to what the library ACTUALLY returns.
func TestAnsweredWithoutActor_IsWhatRemoteProduces(t *testing.T) {

	// Without this, the other tests would pass against a hand-built shape remote no longer produced

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

	// PRECONDITION: remote reports this as a 500 that is not a client error, which is why the
	// poller could not tell it apart from a dead server, and why the status alone is not enough.
	require.Equal(t, 500, derp.ErrorCode(err))
	require.False(t, derp.IsClientError(err))

	// The response that arrived is still in the chain, and it says 200
	status, contentType, answered := answeredWithoutActor(err)
	require.True(t, answered, "remote no longer carries the response status; the rule needs rewriting")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "text/html", contentType)
}

/******************************************
 * Helpers
 ******************************************/

// newPollErrorService returns a Following service that writes into an in-memory collection
func newPollErrorService() (Following, backoffSession, *backoffCollection) {

	collection := &backoffCollection{}
	session := backoffSession{collection: collection}

	// Buffered so the service's own status broadcast does not block the test
	updates := make(chan realtime.Message, 8)

	return Following{sseUpdateChannel: updates}, session, collection
}

// httpStatus builds the derp.HTTPError shape that benpate/remote returns for one status code
func httpStatus(code int) derp.HTTPError {
	return derp.HTTPError{
		Response: derp.HTTPResponseReport{StatusCode: code},
	}
}

// answeredWith builds the error that remote.Transaction.decodeResponseBody returns when a response
// arrives intact but cannot be decoded.
func answeredWith(code int, contentType string) error {

	// WithInternalError below makes the OUTER code 500 while the response underneath says 200

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
